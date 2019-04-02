/*
Copyright 2019 The Kubernetes Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package e2e

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"sigs.k8s.io/federation-v2/test/common"
	"sigs.k8s.io/federation-v2/test/e2e/framework"

	. "github.com/onsi/ginkgo"
)

var _ = Describe("Leader Elector", func() {
	f := framework.NewFederationFramework("leaderelection")
	tl := framework.NewE2ELogger()

	It("should chose secondary instance, primary goes down", func() {
		if !framework.TestContext.TestManagedFederation {
			framework.Skipf("leader election is valid only in managed federation setup")
		}

		const leaderIdentifier = "promoted as leader"

		primaryControllerManager, primaryLogStream, err := spawnControllerManagerProcess(f.KubeConfig().Host, f.FederationSystemNamespace())
		framework.ExpectNoError(err)
		if waitUntilLogStreamContains(tl, primaryLogStream, leaderIdentifier) {
			tl.Log("Primary controller manager became leader")
		} else {
			_ = primaryControllerManager.Process.Kill()
			tl.Fatal("Primary controller manager failed to become leader")
		}

		done := make(chan bool, 1)
		secondaryControllerManager, secondaryLogStream, err := spawnControllerManagerProcess(f.KubeConfig().Host, f.FederationSystemNamespace())
		framework.ExpectNoError(err)
		go func() {
			if waitUntilLogStreamContains(tl, secondaryLogStream, leaderIdentifier) {
				tl.Log("Secondary controller manager became leader")
				done <- true
			} else {
				_ = secondaryControllerManager.Process.Kill()
				tl.Fatal("Secondary controller manager failed to become leader")
			}
		}()

		err = primaryControllerManager.Process.Kill()
		framework.ExpectNoError(err)

		<-done

		err = secondaryControllerManager.Process.Kill()
		framework.ExpectNoError(err)
	})
})

func spawnControllerManagerProcess(master, federationNamespace string) (*exec.Cmd, io.ReadCloser, error) {
	binPath, _ := os.LookupEnv("TEST_ASSET_PATH")
	args := []string{fmt.Sprintf("--master=%s", master),
		fmt.Sprintf("--federation-namespace=%s", federationNamespace),
		"--leader-elect-lease-duration=1500ms",
		"--leader-elect-renew-deadline=1000ms",
		"--leader-elect-retry-period=500ms",
	}
	cmd := exec.Command(binPath+"/controller-manager", args...)

	logStream, err := cmd.StderrPipe()
	if err != nil {
		return nil, nil, err
	}

	if err := cmd.Start(); err != nil {
		return nil, nil, err
	}
	return cmd, logStream, nil
}

func waitUntilLogStreamContains(tl common.TestLogger, stream io.ReadCloser, substr string) bool {
	scanner := bufio.NewScanner(stream)
	done := make(chan bool, 1)
	go func() {
		for scanner.Scan() {
			line := scanner.Text()
			tl.Log(line)
			if strings.Contains(line, substr) {
				done <- true
				return
			}
		}
		done <- false
	}()

	return <-done
}

/*
Copyright 2018 The Kubernetes Authors.

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

package app

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/kubernetes-sigs/kubebuilder/pkg/install"
	"github.com/kubernetes-sigs/kubebuilder/pkg/signals"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	extv1b1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilfeature "k8s.io/apiserver/pkg/util/feature"
	"k8s.io/apiserver/pkg/util/logs"
	"k8s.io/client-go/tools/clientcmd"

	"sigs.k8s.io/federation-v2/cmd/controller-manager/app/leaderelection"
	"sigs.k8s.io/federation-v2/cmd/controller-manager/app/options"
	"sigs.k8s.io/federation-v2/pkg/controller/dnsendpoint"
	"sigs.k8s.io/federation-v2/pkg/controller/federatedcluster"
	"sigs.k8s.io/federation-v2/pkg/controller/federatedtypeconfig"
	"sigs.k8s.io/federation-v2/pkg/controller/ingressdns"
	"sigs.k8s.io/federation-v2/pkg/controller/schedulingmanager"
	"sigs.k8s.io/federation-v2/pkg/controller/servicedns"
	"sigs.k8s.io/federation-v2/pkg/features"
	"sigs.k8s.io/federation-v2/pkg/inject"
	"sigs.k8s.io/federation-v2/pkg/version"
)

var (
	kubeconfig, masterURL string
)

// NewControllerManagerCommand creates a *cobra.Command object with default parameters
func NewControllerManagerCommand() *cobra.Command {
	verFlag := false
	opts := options.NewOptions()

	cmd := &cobra.Command{
		Use: "controller-manager",
		Long: `The Federation controller manager runs a bunch of controllers
which watches federation CRD's and the corresponding resources in federation
member clusters and does the necessary reconciliation`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Fprintf(os.Stdout, "Federation v2 controller-manager version: %s\n", fmt.Sprintf("%#v", version.Get()))
			if verFlag {
				os.Exit(0)
			}
			PrintFlags(cmd.Flags())

			if err := Run(opts); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
				os.Exit(1)
			}
		},
	}

	// Add the command line flags from other dependencies(glog, kubebuilder, etc.)
	cmd.Flags().AddGoFlagSet(flag.CommandLine)

	opts.AddFlags(cmd.Flags())
	cmd.Flags().BoolVar(&verFlag, "version", false, "Prints the Version info of controller-manager")
	cmd.Flags().StringVar(&kubeconfig, "kubeconfig", "", "Path to a kubeconfig. Only required if out-of-cluster.")
	cmd.Flags().StringVar(&masterURL, "master", "", "The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.")

	return cmd
}

// Run runs the controller-manager with options. This should never exit.
func Run(opts *options.Options) error {
	logs.InitLogs()
	defer logs.FlushLogs()

	if err := utilfeature.DefaultFeatureGate.SetFromMap(opts.FeatureGates); err != nil {
		glog.Fatalf("Invalid Feature Gate: %v", err)
	}

	stopChan := signals.SetupSignalHandler()

	var err error
	opts.Config.KubeConfig, err = clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)
	if err != nil {
		panic(err)
	}

	if opts.InstallCRDs {
		if err := install.NewInstaller(opts.Config.KubeConfig).Install(&InstallStrategy{crds: inject.Injector.CRDs}); err != nil {
			glog.Fatalf("Could not create CRDs: %v", err)
		}
	}

	if opts.LimitedScope {
		opts.Config.TargetNamespace = opts.Config.FederationNamespace
		glog.Infof("Federation will be limited to the %q namespace", opts.Config.FederationNamespace)
	} else {
		opts.Config.TargetNamespace = metav1.NamespaceAll
		glog.Info("Federation will target all namespaces")
	}

	elector, err := leaderelection.NewFederationLeaderElector(opts, startControllers)
	if err != nil {
		panic(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		select {
		case <-stopChan:
			cancel()
		case <-ctx.Done():
		}
	}()

	elector.Run(ctx)

	glog.Errorf("lost lease")
	return errors.New("lost lease")
}

func startControllers(opts *options.Options, stopChan <-chan struct{}) {
	if err := federatedcluster.StartClusterController(opts.Config, stopChan, opts.ClusterMonitorPeriod); err != nil {
		glog.Fatalf("Error starting cluster controller: %v", err)
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.SchedulerPreferences) {
		if _, err := schedulingmanager.StartSchedulingManager(opts.Config, stopChan); err != nil {
			glog.Fatalf("Error starting scheduling manager: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.CrossClusterServiceDiscovery) {
		if err := servicedns.StartController(opts.Config, stopChan); err != nil {
			glog.Fatalf("Error starting dns controller: %v", err)
		}

		if err := dnsendpoint.StartServiceDNSEndpointController(opts.Config, stopChan); err != nil {
			glog.Fatalf("Error starting dns endpoint controller: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.FederatedIngress) {
		if err := ingressdns.StartController(opts.Config, stopChan); err != nil {
			glog.Fatalf("Error starting ingress dns controller: %v", err)
		}

		if err := dnsendpoint.StartIngressDNSEndpointController(opts.Config, stopChan); err != nil {
			glog.Fatalf("Error starting ingress dns endpoint controller: %v", err)
		}
	}

	if utilfeature.DefaultFeatureGate.Enabled(features.PushReconciler) {
		if err := federatedtypeconfig.StartController(opts.Config, stopChan); err != nil {
			glog.Fatalf("Error starting federated type config controller: %v", err)
		}
	}
}

type InstallStrategy struct {
	install.EmptyInstallStrategy
	crds []*extv1b1.CustomResourceDefinition
}

func (s *InstallStrategy) GetCRDs() []*extv1b1.CustomResourceDefinition {
	return s.crds
}

// PrintFlags logs the flags in the flagset
func PrintFlags(flags *pflag.FlagSet) {
	flags.VisitAll(func(flag *pflag.Flag) {
		glog.V(1).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
	})
}

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

package federate

import (
	"context"
	"io"

	"github.com/golang/glog"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/rest"

	"sigs.k8s.io/federation-v2/pkg/apis/core/typeconfig"
	fedv1a1 "sigs.k8s.io/federation-v2/pkg/apis/core/v1alpha1"
	genericclient "sigs.k8s.io/federation-v2/pkg/client/generic"
	ctlutil "sigs.k8s.io/federation-v2/pkg/controller/util"
	"sigs.k8s.io/federation-v2/pkg/kubefed2/enable"
	"sigs.k8s.io/federation-v2/pkg/kubefed2/options"
	"sigs.k8s.io/federation-v2/pkg/kubefed2/util"
)

var (
	federate_long = `
		Federate creates a federated resource from a kubernetes resource.
		The target resource must exist in the cluster hosting the federation
		control plane. The control plane must have a FederatedTypeConfig
		for the type of the kubernetes resource. The new federated resource
		will be created with the same name and namespace (if namespaced) as
		the kubernetes resource.

		Current context is assumed to be a Kubernetes cluster hosting
		the federation control plane. Please use the --host-cluster-context
		flag otherwise.`

	federate_example = `
		# Federate resource named "my-dep" in namespace "my-ns" of kubernetes type "deploy"
		kubefed2 federate deploy my-dep -n "my-ns" --host-cluster-context=cluster1`
	// TODO(irfanurrehman): implement —contents flag applicable to namespaces
)

type federateResource struct {
	options.SubcommandOptions
	typeName          string
	resourceName      string
	resourceNamespace string
	output            string
	outputYAML        bool
}

func (j *federateResource) Bind(flags *pflag.FlagSet) {
	flags.StringVarP(&j.resourceNamespace, "namespace", "n", "default", "The namespace of the resource to federate.")
	flags.StringVarP(&j.output, "output", "o", "", "If provided, the resource that would be created in the API by the command is instead output to stdout in the provided format.  Valid format is ['yaml'].")
}

// Complete ensures that options are valid.
func (j *federateResource) Complete(args []string) error {
	if len(args) == 0 {
		return errors.New("TYPE-NAME is required")
	}
	j.typeName = args[0]

	if len(args) == 1 {
		return errors.New("RESOURCE-NAME is required")
	}
	j.resourceName = args[1]

	if j.output == "yaml" {
		j.outputYAML = true
	} else if len(j.output) > 0 {
		return errors.Errorf("Invalid value for --output: %s", j.output)
	}

	return nil
}

// NewCmdFederateResource defines the `federate` command that federates a
// Kubernetes resource of the given kubernetes type.
func NewCmdFederateResource(cmdOut io.Writer, config util.FedConfig) *cobra.Command {
	opts := &federateResource{}

	cmd := &cobra.Command{
		Use:     "federate TYPE-NAME RESOURCE-NAME",
		Short:   "Federate creates a federated resource from a kubernetes resource",
		Long:    federate_long,
		Example: federate_example,
		Run: func(cmd *cobra.Command, args []string) {
			err := opts.Complete(args)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}

			err = opts.Run(cmdOut, config)
			if err != nil {
				glog.Fatalf("error: %v", err)
			}
		},
	}

	flags := cmd.Flags()
	opts.CommonBind(flags)
	opts.Bind(flags)

	return cmd
}

// Run is the implementation of the `federate resource` command.
func (j *federateResource) Run(cmdOut io.Writer, config util.FedConfig) error {
	hostConfig, err := config.HostConfig(j.HostClusterContext, j.Kubeconfig)
	if err != nil {
		return errors.Wrap(err, "Failed to get host cluster config")
	}

	qualifiedTypeName := ctlutil.QualifiedName{
		Namespace: j.FederationNamespace,
		Name:      j.typeName,
	}

	qualifiedResourceName := ctlutil.QualifiedName{
		Namespace: j.resourceNamespace,
		Name:      j.resourceName,
	}

	resources, err := GetFedResources(hostConfig, qualifiedTypeName, qualifiedResourceName)
	if err != nil {
		return err
	}

	if j.outputYAML {
		err := util.WriteUnstructuredToYaml(resources.FedResource, cmdOut)
		if err != nil {
			return errors.Wrap(err, "Failed to write federated resource to YAML")
		}
		return nil
	}

	return CreateFedResource(hostConfig, resources, j.DryRun)
}

type fedResources struct {
	typeConfig  typeconfig.Interface
	FedResource *unstructured.Unstructured
}

func GetFedResources(hostConfig *rest.Config, qualifiedTypeName, qualifiedName ctlutil.QualifiedName) (*fedResources, error) {
	typeConfig, err := lookupTypeDetails(hostConfig, qualifiedTypeName)
	if err != nil {
		return nil, err
	}

	targetResource, err := getTargetResource(hostConfig, typeConfig, qualifiedName)
	if err != nil {
		return nil, err
	}

	fedResource, err := FederatedResourceFromTargetResource(typeConfig, targetResource)
	if err != nil {
		return nil, errors.Wrapf(err, "Error getting %s from %s %q", typeConfig.GetFederatedType().Kind, typeConfig.GetTarget().Kind, qualifiedName)
	}

	return &fedResources{
		typeConfig:  typeConfig,
		FedResource: fedResource,
	}, nil
}

func lookupTypeDetails(hostConfig *rest.Config, typeName ctlutil.QualifiedName) (*fedv1a1.FederatedTypeConfig, error) {
	apiResource, err := enable.LookupAPIResource(hostConfig, typeName.Name, "")
	if err != nil {
		return nil, err
	}
	typeName.Name = typeconfig.GroupQualifiedName(*apiResource)

	client, err := genericclient.New(hostConfig)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to get federation client")
	}

	typeConfig := &fedv1a1.FederatedTypeConfig{}
	err = client.Get(context.TODO(), typeConfig, typeName.Namespace, typeName.Name)
	if apierrors.IsNotFound(err) {
		return nil, errors.Wrapf(err, "FederatedTypeConfig %q not found. Try 'kubefed2 enable type %s' before federating the resource", typeName, typeName.Name)
	}
	if err != nil {
		return nil, errors.Wrapf(err, "Error retrieving FederatedTypeConfig %q", typeName)
	}

	glog.V(2).Infof("API Resource and FederatedTypeConfig found for %q", typeName)
	return typeConfig, nil
}

func getTargetResource(hostConfig *rest.Config, typeConfig *fedv1a1.FederatedTypeConfig, qualifiedName ctlutil.QualifiedName) (*unstructured.Unstructured, error) {
	targetAPIResource := typeConfig.GetTarget()
	targetClient, err := ctlutil.NewResourceClient(hostConfig, &targetAPIResource)
	if err != nil {
		return nil, errors.Wrapf(err, "Error creating client for %s", targetAPIResource.Kind)
	}

	kind := targetAPIResource.Kind
	resource, err := targetClient.Resources(qualifiedName.Namespace).Get(qualifiedName.Name, metav1.GetOptions{})
	if err != nil {
		return nil, errors.Wrapf(err, "Error retrieving target %s %q", kind, qualifiedName)
	}

	glog.V(2).Infof("Target %s %q found", kind, qualifiedName)
	return resource, nil
}

func FederatedResourceFromTargetResource(typeConfig typeconfig.Interface, targetResource *unstructured.Unstructured) (*unstructured.Unstructured, error) {
	fedAPIResource := typeConfig.GetFederatedType()

	// Special handling is needed for some controller set fields.
	if typeConfig.GetTarget().Kind == ctlutil.ServiceAccountKind {
		unstructured.RemoveNestedField(targetResource.Object, ctlutil.SecretsField)
	}

	if typeConfig.GetTarget().Kind == ctlutil.ServiceKind {
		var targetPorts []interface{}
		targetPorts, ok, err := unstructured.NestedSlice(targetResource.Object, "spec", "ports")
		if err != nil {
			return nil, err
		}
		if ok {
			for index := range targetPorts {
				port := targetPorts[index].(map[string]interface{})
				delete(port, "nodePort")
				targetPorts[index] = port
			}
			err := unstructured.SetNestedSlice(targetResource.Object, targetPorts, "spec", "ports")
			if err != nil {
				return nil, err
			}
		}
		unstructured.RemoveNestedField(targetResource.Object, "spec", "clusterIP")
	}

	qualifiedName := ctlutil.NewQualifiedName(targetResource)
	resourceNamespace := getNamespace(typeConfig, qualifiedName)
	fedResource := &unstructured.Unstructured{}
	SetBasicMetaFields(fedResource, fedAPIResource, qualifiedName.Name, resourceNamespace, "")
	RemoveUnwantedFields(targetResource)

	err := unstructured.SetNestedField(fedResource.Object, targetResource.Object, ctlutil.SpecField, ctlutil.TemplateField)
	if err != nil {
		return nil, err
	}
	err = unstructured.SetNestedStringMap(fedResource.Object, map[string]string{}, ctlutil.SpecField, ctlutil.PlacementField, ctlutil.ClusterSelectorField, ctlutil.MatchLabelsField)
	if err != nil {
		return nil, err
	}

	return fedResource, err
}

func getNamespace(typeConfig typeconfig.Interface, qualifiedName ctlutil.QualifiedName) string {
	if typeConfig.GetTarget().Kind == ctlutil.NamespaceKind {
		return qualifiedName.Name
	}
	return qualifiedName.Namespace
}

func CreateFedResource(hostConfig *rest.Config, resources *fedResources, dryrun bool) error {
	if resources.typeConfig.GetTarget().Kind == ctlutil.NamespaceKind {
		// TODO: irfanurrehman: Can a target namespace be federated into another namespace?
		glog.Infof("Resource to federate is a namespace. Given namespace will itself be the container for the federated namespace")
	}

	fedAPIResource := resources.typeConfig.GetFederatedType()
	fedKind := fedAPIResource.Kind
	fedClient, err := ctlutil.NewResourceClient(hostConfig, &fedAPIResource)
	if err != nil {
		return errors.Wrapf(err, "Error creating client for %s", fedKind)
	}

	qualifiedFedName := ctlutil.NewQualifiedName(resources.FedResource)
	if !dryrun {
		_, err = fedClient.Resources(resources.FedResource.GetNamespace()).Create(resources.FedResource, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrapf(err, "Error creating %s %q", fedKind, qualifiedFedName)
		}
	}

	glog.Infof("Successfully created %s %q from %s", fedKind, qualifiedFedName, resources.typeConfig.GetTarget().Kind)
	return nil
}

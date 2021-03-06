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

package kubefnord

import (
	"fmt"
	"io"
	"time"

	"github.com/golang/glog"

	fedv1a1 "github.com/kubernetes-sigs/federation-v2/pkg/apis/core/v1alpha1"
	fedclient "github.com/kubernetes-sigs/federation-v2/pkg/client/clientset/versioned"
	controllerutil "github.com/kubernetes-sigs/federation-v2/pkg/controller/util"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefnord/options"
	"github.com/kubernetes-sigs/federation-v2/pkg/kubefnord/util"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	client "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	crv1a1 "k8s.io/cluster-registry/pkg/apis/clusterregistry/v1alpha1"
	crclient "k8s.io/cluster-registry/pkg/client/clientset/versioned"
)

const (
	serviceAccountSecretTimeout = 30 * time.Second
)

var (
	join_long = `
		Join adds a cluster to a federation.

		Current context is assumed to be a Kubernetes cluster
		with an aggregated federation API server. Please use
		the --host-cluster-context flag otherwise.`
	join_example = `
		# Join a cluster to a federation by specifying the
		# cluster name and the context name of the federation
		# control plane's host cluster. Cluster name must be
		# a valid RFC 1123 subdomain name. Cluster context
		# must be specified if the cluster name is different
		# than the cluster's context in the local kubeconfig.
		kubefnord join foo --host-cluster-context=bar`
)

type joinFederation struct {
	options.SubcommandOptions
	joinFederationOptions
}

type joinFederationOptions struct {
	clusterContext string
	secretName     string
	addToRegistry  bool
}

// Bind adds the join specific arguments to the flagset passed in as an
// argument.
func (o *joinFederationOptions) Bind(flags *pflag.FlagSet) {
	flags.StringVar(&o.clusterContext, "cluster-context", "",
		"Name of the cluster's context in the local kubeconfig. Defaults to cluster name if unspecified.")
	flags.StringVar(&o.secretName, "secret-name", "",
		"Name of the secret where the cluster's credentials will be stored in the host cluster. This name should be a valid RFC 1035 label. If unspecified, defaults to a generated name containing the cluster name.")
	flags.BoolVar(&o.addToRegistry, "add-to-registry", false,
		"Add the cluster to the cluster registry that is aggregated with the kubernetes API server running in the host cluster context.")
}

// NewCmdJoin defines the `join` command that joins a cluster to a
// federation.
func NewCmdJoin(cmdOut io.Writer, config util.FedConfig) *cobra.Command {
	opts := &joinFederation{}

	cmd := &cobra.Command{
		Use:     "join CLUSTER_NAME --host-cluster-context=HOST_CONTEXT",
		Short:   "Join a cluster to a federation",
		Long:    join_long,
		Example: join_example,
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

// Complete ensures that options are valid and marshals them if necessary.
func (j *joinFederation) Complete(args []string) error {
	err := j.SetName(args)
	if err != nil {
		return err
	}

	if j.clusterContext == "" {
		glog.V(2).Infof("Defaulting cluster context to joining cluster name %s", j.ClusterName)
		j.clusterContext = j.ClusterName
	}

	glog.V(2).Infof("Args and flags: name %s, host: %s, host-system-namespace: %s, kubeconfig: %s, cluster-context: %s, secret-name: %s, dry-run: %v",
		j.ClusterName, j.Host, j.FederationNamespace, j.Kubeconfig, j.clusterContext,
		j.secretName, j.DryRun)

	return nil
}

// Run is the implementation of the `join federation` command.
func (j *joinFederation) Run(cmdOut io.Writer, config util.FedConfig) error {
	hostConfig, err := config.HostConfig(j.Host, j.Kubeconfig)
	if err != nil {
		// TODO(font): Return new error with this same text so it can be output
		// by caller.
		glog.V(2).Infof("Failed to get host cluster config: %v", err)
		return err
	}

	clusterConfig, err := config.ClusterConfig(j.clusterContext, j.Kubeconfig)
	if err != nil {
		glog.V(2).Infof("Failed to get joining cluster config: %v", err)
		return err
	}

	err = JoinCluster(hostConfig, clusterConfig, j.FederationNamespace,
		j.Host, j.ClusterName, j.secretName, j.addToRegistry, j.DryRun)
	if err != nil {
		return err
	}

	return nil
}

// JoinCluster performs all the necessary steps to join a cluster to the
// federation provided the required set of parameters are passed in.
func JoinCluster(hostConfig, clusterConfig *rest.Config, federationNamespace,
	host, joiningClusterName, secretName string, addToRegistry, dryRun bool) error {
	hostClientset, err := util.HostClientset(hostConfig)
	if err != nil {
		glog.V(2).Infof("Failed to get host cluster clientset: %v", err)
		return err
	}

	clusterClientset, err := util.ClusterClientset(clusterConfig)
	if err != nil {
		glog.V(2).Infof("Failed to get joining cluster clientset: %v", err)
		return err
	}

	fedClientset, err := util.FedClientset(hostConfig)
	if err != nil {
		glog.V(2).Infof("Failed to get federation clientset: %v", err)
		return err
	}

	glog.V(2).Infof("Performing preflight checks.")
	err = performPreflightChecks(clusterClientset, joiningClusterName, host, federationNamespace)
	if err != nil {
		return err
	}

	if addToRegistry {
		err = addToClusterRegistry(hostConfig, clusterConfig.Host, joiningClusterName, dryRun)
		if err != nil {
			return err
		}
	} else {
		// TODO(font): If cluster exists in clusterregistry, grab the
		// ServerAddress from the KubernetesAPIEndpoints to create a
		// clusterClientset from it.
		err = verifyExistsInClusterRegistry(hostConfig, joiningClusterName)
		if err != nil {
			return err
		}
	}

	glog.V(2).Infof("Creating %s namespace in joining cluster", federationNamespace)
	_, err = createFederationNamespace(clusterClientset, federationNamespace,
		joiningClusterName, dryRun)
	if err != nil {
		glog.V(2).Infof("Error creating %s namespace in joining cluster: %v",
			federationNamespace, err)
		return err
	}
	glog.V(2).Infof("Created %s namespace in joining cluster", federationNamespace)

	// Create a service account and use its credentials.
	glog.V(2).Info("Creating cluster credentials secret")

	secret, err := createRBACSecret(hostClientset, clusterClientset,
		federationNamespace, joiningClusterName, host,
		secretName, dryRun)
	if err != nil {
		glog.V(2).Infof("Could not create cluster credentials secret: %v", err)
		return err
	}

	glog.V(2).Info("Cluster credentials secret created")

	glog.V(2).Info("Creating federated cluster resource")

	_, err = createFederatedCluster(fedClientset, joiningClusterName, secret.Name, dryRun)
	if err != nil {
		glog.V(2).Infof("Failed to create federated cluster resource: %v", err)
		return err
	}

	glog.V(2).Info("Created federated cluster resource")
	return nil
}

// performPreflightChecks checks that the host and joining clusters are in
// a consistent state.
func performPreflightChecks(clusterClientset client.Interface, name, host,
	federationNamespace string) error {
	// Make sure there is no existing service account in the joining cluster.
	saName := util.ClusterServiceAccountName(name, host)
	sa, err := clusterClientset.CoreV1().ServiceAccounts(federationNamespace).Get(saName,
		metav1.GetOptions{})

	if errors.IsNotFound(err) {
		return nil
	} else if err != nil {
		return err
	} else if sa != nil {
		return fmt.Errorf("service account already exists in joining cluster")
	}

	return nil
}

// addToClusterRegistry handles adding the cluster to the cluster registry and
// reports progress.
func addToClusterRegistry(hostConfig *rest.Config, host, joiningClusterName string,
	dryRun bool) error {
	// Get the cluster registry clientset using the host cluster config.
	crClientset, err := util.ClusterRegistryClientset(hostConfig)
	if err != nil {
		glog.V(2).Infof("Failed to get cluster registry clientset: %v", err)
		return err
	}

	glog.V(2).Info("Registering cluster with the cluster registry.")

	_, err = registerCluster(crClientset, host, joiningClusterName, dryRun)
	if err != nil {
		glog.V(2).Infof("Could not register cluster with the cluster registry: %v", err)
		return err
	}

	glog.V(2).Info("Registered cluster with the cluster registry.")
	return nil
}

// verifyExistsInClusterRegistry verifies that the given joining cluster name exists
// in the cluster registry.
func verifyExistsInClusterRegistry(hostConfig *rest.Config, joiningClusterName string) error {
	// Get the cluster registry clientset using the host cluster config.
	crClientset, err := util.ClusterRegistryClientset(hostConfig)
	if err != nil {
		glog.V(2).Infof("Failed to get cluster registry clientset: %v", err)
		return err
	}

	glog.V(2).Infof("Verifying cluster %s exists in the cluster registry.",
		joiningClusterName)

	_, err = crClientset.ClusterregistryV1alpha1().Clusters(controllerutil.MulticlusterPublicNamespace).Get(joiningClusterName,
		metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return fmt.Errorf("Cluster %s does not exist in the cluster registry.",
				joiningClusterName)
		}

		glog.V(2).Infof("Could not retrieve cluster %s from the cluster registry: %v",
			joiningClusterName, err)
		return err
	}

	glog.V(2).Infof("Verified cluster %s exists in the cluster registry.", joiningClusterName)
	return nil
}

// registerCluster registers a cluster with the cluster registry.
// TODO: save off service account authinfo for cluster.
func registerCluster(crClientset *crclient.Clientset, host, joiningClusterName string,
	dryRun bool) (*crv1a1.Cluster, error) {
	cluster := &crv1a1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: joiningClusterName,
		},
		Spec: crv1a1.ClusterSpec{
			KubernetesAPIEndpoints: crv1a1.KubernetesAPIEndpoints{
				ServerEndpoints: []crv1a1.ServerAddressByClientCIDR{
					{
						ClientCIDR:    "0.0.0.0/0",
						ServerAddress: host,
					},
				},
			},
		},
	}

	if dryRun {
		return cluster, nil
	}

	cluster, err := crClientset.ClusterregistryV1alpha1().Clusters(controllerutil.MulticlusterPublicNamespace).Create(cluster)
	if err != nil {
		return cluster, err
	}

	return cluster, nil
}

// createFederatedCluster creates a federated cluster resource that associates
// the cluster and secret.
func createFederatedCluster(fedClientset *fedclient.Clientset, joiningClusterName,
	secretName string, dryRun bool) (*fedv1a1.FederatedCluster, error) {
	fedCluster := &fedv1a1.FederatedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: joiningClusterName,
		},
		Spec: fedv1a1.FederatedClusterSpec{
			ClusterRef: corev1.LocalObjectReference{
				Name: joiningClusterName,
			},
			SecretRef: &corev1.LocalObjectReference{
				Name: secretName,
			},
		},
	}

	if dryRun {
		return fedCluster, nil
	}

	fedCluster, err := fedClientset.CoreV1alpha1().FederatedClusters(controllerutil.FederationSystemNamespace).Create(fedCluster)

	if err != nil {
		return fedCluster, err
	}

	return fedCluster, nil
}

// createFederationNamespace creates the federation namespace in the cluster
// associated with clusterClientset, if it doesn't already exist.
func createFederationNamespace(clusterClientset client.Interface, federationNamespace,
	joiningClusterName string, dryRun bool) (*corev1.Namespace, error) {
	federationNS := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: federationNamespace,
		},
	}

	if dryRun {
		return federationNS, nil
	}

	_, err := clusterClientset.CoreV1().Namespaces().Create(federationNS)
	if err != nil && !errors.IsAlreadyExists(err) {
		glog.V(2).Infof("Could not create %s namespace: %v", federationNamespace, err)
		return nil, err
	}
	return federationNS, nil
}

// createRBACSecret creates a secret in the joining cluster using a service
// account, and populate that secret into the host cluster to allow it to
// access the joining cluster.
func createRBACSecret(hostClusterClientset, joiningClusterClientset client.Interface,
	namespace, joiningClusterName, hostClusterName,
	secretName string, dryRun bool) (*corev1.Secret, error) {

	glog.V(2).Info("Creating service account in joining cluster")

	saName, err := createServiceAccount(joiningClusterClientset, namespace,
		joiningClusterName, hostClusterName, dryRun)
	if err != nil {
		glog.V(2).Infof("Error creating service account in joining cluster: %v", err)
		return nil, err
	}

	glog.V(2).Infof("Created service account in joining cluster")

	glog.V(2).Info("Creating role binding for service account in joining cluster")

	_, err = createClusterRoleBinding(joiningClusterClientset, saName, namespace,
		joiningClusterName, dryRun)
	if err != nil {
		glog.V(2).Infof("Error creating role binding for service account in joining cluster: %v",
			err)
		return nil, err
	}

	glog.V(2).Info("Created role binding for service account in joining cluster")

	glog.V(2).Info("Creating secret in host cluster")

	secret, err := populateSecretInHostCluster(joiningClusterClientset, hostClusterClientset,
		saName, namespace, joiningClusterName, secretName, dryRun)
	if err != nil {
		glog.V(2).Infof("Error creating secret in host cluster: %v", err)
		return nil, err
	}

	glog.V(2).Info("Created secret in host cluster")

	return secret, nil
}

// createServiceAccount creates a service account in the cluster associated
// with clusterClientset with credentials that will be used by the host cluster
// to access its API server.
func createServiceAccount(clusterClientset client.Interface, namespace,
	joiningClusterName, hostClusterName string, dryRun bool) (string, error) {
	saName := util.ClusterServiceAccountName(joiningClusterName, hostClusterName)
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      saName,
			Namespace: namespace,
		},
	}

	if dryRun {
		return saName, nil
	}

	// Create a new service account.
	_, err := clusterClientset.CoreV1().ServiceAccounts(namespace).Create(sa)
	if err != nil {
		return "", err
	}

	return saName, nil
}

// createClusterRoleBinding creates an RBAC cluster role and binding that
// allows the service account identified by saName to access all resources in
// all namespaces in the cluster associated with clusterClientset.
func createClusterRoleBinding(clusterClientset client.Interface, saName, namespace,
	joiningClusterName string, dryRun bool) (*rbacv1.ClusterRoleBinding, error) {

	roleName := util.ClusterRoleName(saName)

	rules := []rbacv1.PolicyRule{
		{
			Verbs:     []string{rbacv1.VerbAll},
			APIGroups: []string{rbacv1.APIGroupAll},
			Resources: []string{rbacv1.ResourceAll},
		},
		{
			Verbs:           []string{"Get"},
			NonResourceURLs: []string{"/healthz"},
		},
	}

	clusterRole := &rbacv1.ClusterRole{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespace,
		},
		Rules: rules,
	}

	// TODO: This should limit its access to only necessary resources.
	clusterRoleBinding := rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      roleName,
			Namespace: namespace,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      rbacv1.ServiceAccountKind,
				Name:      saName,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: rbacv1.GroupName,
			Kind:     "ClusterRole",
			Name:     roleName,
		},
	}

	if dryRun {
		return &clusterRoleBinding, nil
	}

	_, err := clusterClientset.RbacV1().ClusterRoles().Create(clusterRole)
	if err != nil {
		glog.V(2).Infof("Could not create cluster role for service account in joining cluster: %v",
			err)
		return nil, err
	}

	_, err = clusterClientset.RbacV1().ClusterRoleBindings().Create(&clusterRoleBinding)
	if err != nil {
		glog.V(2).Infof("Could not create cluster role binding for service account in joining cluster: %v",
			err)
		return nil, err
	}

	return &clusterRoleBinding, nil
}

// populateSecretInHostCluster copies the service account secret for saName
// from the cluster referenced by clusterClientset to the client referenced by
// hostClientset, putting it in a secret named secretName in the provided
// namespace.
func populateSecretInHostCluster(clusterClientset, hostClientset client.Interface,
	saName, namespace, joiningClusterName, secretName string,
	dryRun bool) (*corev1.Secret, error) {
	if dryRun {
		dryRunSecret := &corev1.Secret{}
		dryRunSecret.Name = secretName
		return dryRunSecret, nil
	}

	// Get the secret from the joining cluster.
	var secret *corev1.Secret
	err := wait.PollImmediate(1*time.Second, serviceAccountSecretTimeout, func() (bool, error) {
		sa, err := clusterClientset.CoreV1().ServiceAccounts(namespace).Get(saName,
			metav1.GetOptions{})
		if err != nil {
			return false, nil
		}

		for _, objReference := range sa.Secrets {
			saSecretName := objReference.Name
			var err error
			secret, err = clusterClientset.CoreV1().Secrets(namespace).Get(saSecretName,
				metav1.GetOptions{})
			if err != nil {
				return false, nil
			}
			if secret.Type == corev1.SecretTypeServiceAccountToken {
				glog.V(2).Infof("Using secret named: %s", secret.Name)
				return true, nil
			}
		}
		return false, nil
	})

	if err != nil {
		glog.V(2).Infof("Could not get service account secret from joining cluster: %v", err)
		return nil, err
	}

	// Create a parallel secret in the host cluster.
	v1Secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
		},
		Data: secret.Data,
	}

	if secretName == "" {
		v1Secret.GenerateName = joiningClusterName + "-"
	} else {
		v1Secret.Name = secretName
	}

	v1SecretResult, err := hostClientset.CoreV1().Secrets(namespace).Create(&v1Secret)
	if err != nil {
		glog.V(2).Infof("Could not create secret in host cluster: %v", err)
		return nil, err
	}

	glog.V(2).Infof("Created secret in host cluster named: %s", v1SecretResult.Name)
	return v1SecretResult, nil
}

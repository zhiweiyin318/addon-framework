package main

import (
	"context"
	"embed"
	"os"

	"github.com/openshift/library-go/pkg/assets"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"open-cluster-management.io/addon-framework/pkg/agent"
	"open-cluster-management.io/addon-framework/pkg/utils"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

const defaultExampleImage = "quay.io/open-cluster-management/helloworld-addon:latest"

//go:embed manifests
var fs embed.FS

var manifestFiles = []string{
	// serviceaccount to run addon-agent
	"manifests/serviceaccount.yaml",
	// clusterrolebinding to bind appropriate clusterrole to the serviceaccount
	"manifests/clusterrolebinding.yaml",
	// deployment to deploy addon-agent
	"manifests/deployment.yaml",
}

var agentPermissionFiles = []string{
	// role with RBAC rules to access resources on hub
	"manifests/role.yaml",
	// rolebinding to bind the above role to a certain user group
	"manifests/rolebinding.yaml",
}

// the Reserved template config fields (ClusterName,AddonInstallNamespace and KubeConfigSecret) are not mandatory here.
// these fields can also be overridden by the TemplateConfig field in the annotation of addon cr.
type templateConfig struct {
	ClusterName           string
	AddonInstallNamespace string
	KubeConfigSecret      string
	Image                 string
}

func newTemplateConfig() templateConfig {
	image := os.Getenv("EXAMPLE_IMAGE_NAME")
	if len(image) == 0 {
		image = defaultExampleImage
	}
	return templateConfig{Image: image}
}

func newRegistrationOption(kubeConfig *rest.Config, recorder events.Recorder, agentName string) *agent.RegistrationOption {
	return &agent.RegistrationOption{
		CSRConfigurations: agent.KubeClientSignerConfigurations("helloworld", agentName),
		CSRApproveCheck:   utils.DefaultCSRApprover(agentName),
		PermissionConfig: func(cluster *clusterv1.ManagedCluster, addon *addonapiv1alpha1.ManagedClusterAddOn) error {
			kubeclient, err := kubernetes.NewForConfig(kubeConfig)
			if err != nil {
				return err
			}

			for _, file := range agentPermissionFiles {
				if err := applyManifestFromFile(file, cluster.Name, addon.Name, kubeclient, recorder); err != nil {
					return err
				}
			}

			return nil
		},
	}
}

func applyManifestFromFile(file, clusterName, addonName string, kubeclient *kubernetes.Clientset, recorder events.Recorder) error {
	groups := agent.DefaultGroups(clusterName, addonName)
	config := struct {
		ClusterName string
		Group       string
	}{
		ClusterName: clusterName,
		Group:       groups[0],
	}

	results := resourceapply.ApplyDirectly(context.Background(),
		resourceapply.NewKubeClientHolder(kubeclient),
		recorder,
		func(name string) ([]byte, error) {
			template, err := fs.ReadFile(file)
			if err != nil {
				return nil, err
			}
			return assets.MustCreateAssetFromTemplate(name, template, config).Data, nil
		},
		file,
	)

	for _, result := range results {
		if result.Error != nil {
			return result.Error
		}
	}

	return nil
}

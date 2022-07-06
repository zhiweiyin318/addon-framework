package helloworld

import (
	"embed"
	"fmt"
	"os"

	helloworldagent "github.com/zhiweiyin318/addon-framework/examples/helloworld/agent"
	"github.com/zhiweiyin318/addon-framework/examples/rbac"
	"github.com/zhiweiyin318/addon-framework/pkg/addonfactory"
	"github.com/zhiweiyin318/addon-framework/pkg/agent"
	"github.com/zhiweiyin318/addon-framework/pkg/utils"
	"k8s.io/client-go/rest"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

const (
	defaultExampleImage = "quay.io/open-cluster-management/helloworld-addon:latest"
	AddonName           = "helloworld"
)

//go:embed manifests
//go:embed manifests/templates
var FS embed.FS

func NewRegistrationOption(kubeConfig *rest.Config, agentName string) *agent.RegistrationOption {
	return &agent.RegistrationOption{
		CSRConfigurations: agent.KubeClientSignerConfigurations(AddonName, agentName),
		CSRApproveCheck:   utils.DefaultCSRApprover(agentName),
		PermissionConfig:  rbac.AddonRBAC(kubeConfig),
	}
}

func GetValues(cluster *clusterv1.ManagedCluster,
	addon *addonapiv1alpha1.ManagedClusterAddOn) (addonfactory.Values, error) {
	installNamespace := addon.Spec.InstallNamespace
	if len(installNamespace) == 0 {
		installNamespace = helloworldagent.HelloworldAgentInstallationNamespace
	}

	image := os.Getenv("EXAMPLE_IMAGE_NAME")
	if len(image) == 0 {
		image = defaultExampleImage
	}

	manifestConfig := struct {
		KubeConfigSecret      string
		ClusterName           string
		AddonInstallNamespace string
		Image                 string
	}{
		KubeConfigSecret:      fmt.Sprintf("%s-hub-kubeconfig", addon.Name),
		AddonInstallNamespace: installNamespace,
		ClusterName:           cluster.Name,
		Image:                 image,
	}

	return addonfactory.StructToValues(manifestConfig), nil
}

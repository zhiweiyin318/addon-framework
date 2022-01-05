package main

import (
	"fmt"
	"testing"

	"github.com/openshift/library-go/pkg/assets"
)

func TestTemplate(t *testing.T) {
	config := newTemplateConfig()
	config.NodeSelector = map[string]string{"host": "ssd"}
	config.ClusterName = "cluster1"
	config.IsHubCluster = false
	config.IsCRDV1beta1 = true
	config.HTTPSProxy = "http://1.1.1.1"
	config.HTTPProxy = "http://1.1.1.1"
	config.NOProxy = "localhost"
	config.KubeConfigSecret = fmt.Sprintf("%s-hub-kubeconfig", addonName)
	config.AddonInstallNamespace = "open-cluster-management-agent-addon"

	for _, file := range manifestFiles {
		template, err := fs.ReadFile(file)
		if err != nil {
			t.Errorf("failed to read files %v", err)
		}

		err = assets.MustCreateAssetFromTemplate(file, template, &config).WriteFile("./render")
		if err != nil {
			t.Errorf("failed to write files %v", err)
		}
	}
}

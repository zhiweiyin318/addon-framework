package addonfactory

import (
	"embed"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
	clusterv1apha1 "open-cluster-management.io/api/cluster/v1alpha1"
)

//go:embed testmanifests
var fs embed.FS

func newManagedCluster(name string) *clusterv1.ManagedCluster {
	return &clusterv1.ManagedCluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: clusterv1.ManagedClusterSpec{},
	}
}

func newManagedClusterAddon(
	name, clusterName,
	installNamespace string,
	annotations map[string]string) *addonapiv1alpha1.ManagedClusterAddOn {
	return &addonapiv1alpha1.ManagedClusterAddOn{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   clusterName,
			Annotations: annotations,
		},
		Spec: addonapiv1alpha1.ManagedClusterAddOnSpec{InstallNamespace: installNamespace},
	}
}

func TestDefaultAgentAddon_Manifests(t *testing.T) {
	type config struct {
		NodeSelector map[string]string
		Image        string
	}

	scheme := runtime.NewScheme()
	_ = clusterv1apha1.Install(scheme)

	cases := []struct {
		name                     string
		files                    []string
		scheme                   *runtime.Scheme
		clusterName              string
		addonName                string
		installNamespace         string
		userConfig               config
		annotationConfig         string
		expectedInstallNamespace string
		expectedNodeSelector     map[string]string
		expectedImage            string
	}{
		{
			name:                     "template render ok with annotation config and default scheme",
			files:                    []string{"testmanifests/deployment.yaml"},
			clusterName:              "cluster1",
			addonName:                "helloworld",
			installNamespace:         "myNs",
			userConfig:               config{Image: "quay.io/helloworld:latest"},
			annotationConfig:         `{"NodeSelector":{"host":"ssd"},"Image":"quay.io/helloworld:2.4"}`,
			expectedInstallNamespace: "myNs",
			expectedNodeSelector:     map[string]string{"host": "ssd"},
			expectedImage:            "quay.io/helloworld:2.4",
		},
		{
			name:                     "deployment template render ok with default scheme but no annotation config",
			files:                    []string{"testmanifests/deployment.yaml"},
			clusterName:              "cluster1",
			addonName:                "helloworld",
			userConfig:               config{Image: "quay.io/helloworld:latest"},
			expectedInstallNamespace: AddonDefaultInstallNamespace,
			expectedNodeSelector:     map[string]string{},
			expectedImage:            "quay.io/helloworld:latest",
		},
		{
			name:                     "deployment template render ok with default scheme,but no userConfig",
			files:                    []string{"testmanifests/deployment.yaml"},
			clusterName:              "cluster1",
			addonName:                "helloworld",
			annotationConfig:         `{"NodeSelector":{"host":"ssd"},"Image":"quay.io/helloworld:2.4"}`,
			expectedInstallNamespace: AddonDefaultInstallNamespace,
			expectedNodeSelector:     map[string]string{"host": "ssd"},
			expectedImage:            "quay.io/helloworld:2.4",
		},
		{
			name:                     "template render ok with userConfig and custom scheme",
			files:                    []string{"testmanifests/clusterclaim.yaml"},
			scheme:                   scheme,
			clusterName:              "cluster1",
			addonName:                "helloworld",
			annotationConfig:         `{"NodeSelector":{"host":"ssd"},"Image":"quay.io/helloworld:2.4"}`,
			expectedInstallNamespace: AddonDefaultInstallNamespace,
			expectedNodeSelector:     map[string]string{"host": "ssd"},
			expectedImage:            "quay.io/helloworld:2.4",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			cluster := newManagedCluster(c.clusterName)
			clusterAddon := newManagedClusterAddon(c.addonName, c.clusterName, c.installNamespace,
				map[string]string{templateConfigName: c.annotationConfig})

			agentAddon, err := NewAgentAddonFactoryWithTemplates(c.addonName, fs, c.files...).
				WithScheme(c.scheme).WithTemplateConfig(c.userConfig).Build()
			if err != nil {
				t.Errorf("expected no error, got err %v", err)
			}
			objects, err := agentAddon.Manifests(cluster, clusterAddon)
			if err != nil {
				t.Errorf("expected no error, got err %v", err)
			}
			for _, o := range objects {
				switch object := o.(type) {
				case *appsv1.Deployment:
					if object.Namespace != c.expectedInstallNamespace {
						t.Errorf("expected namespace is %s, but got %s", c.expectedInstallNamespace, object.Namespace)
					}

					labels := object.GetLabels()
					if labels["clusterName"] != c.clusterName {
						t.Errorf("expected label is %s, but got %s", c.clusterName, labels["clusterName"])
					}

					nodeSelector := object.Spec.Template.Spec.NodeSelector
					for k, v := range c.expectedNodeSelector {
						if nodeSelector[k] != v {
							t.Errorf("expected nodeSelector is %v, but got %v", c.expectedNodeSelector, nodeSelector)
						}
					}

					if object.Spec.Template.Spec.Containers[0].Image != c.expectedImage {
						t.Errorf("expected image is %s, but got %s", c.expectedImage, object.Spec.Template.Spec.Containers[0].Image)
					}
				case *clusterv1apha1.ClusterClaim:
					if object.GetName() != c.expectedInstallNamespace {
						t.Errorf("expected name is %s, but got %s", c.expectedInstallNamespace, object.GetName())
					}
					if object.Spec.Value != c.expectedImage {
						t.Errorf("expected image is %s, but got %s", c.expectedImage, object.Spec.Value)
					}
				}

			}
		})
	}
}

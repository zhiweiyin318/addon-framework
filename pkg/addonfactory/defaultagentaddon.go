package addonfactory

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/openshift/library-go/pkg/assets"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/klog/v2"
	"open-cluster-management.io/addon-framework/pkg/agent"
	addonapiv1alpha1 "open-cluster-management.io/api/addon/v1alpha1"
	clusterv1 "open-cluster-management.io/api/cluster/v1"
)

// customized template config name in annotation
const templateConfigName string = "templateConfig"

// the reserved fields for template config
const (
	clusterName           string = "ClusterName"
	addonInstallNamespace string = "AddonInstallNamespace"
	kubeConfigSecret      string = "KubeConfigSecret"
)

type DefaultAgentAddon struct {
	decoder           runtime.Decoder
	templateData      map[string][]byte
	templateConfig    interface{}
	agentAddonOptions agent.AgentAddonOptions
}

func newDefaultAgentAddon(
	scheme *runtime.Scheme,
	templateConfig interface{},
	agentAddonOptions agent.AgentAddonOptions) *DefaultAgentAddon {
	return &DefaultAgentAddon{
		decoder:           serializer.NewCodecFactory(scheme).UniversalDeserializer(),
		templateData:      map[string][]byte{},
		templateConfig:    templateConfig,
		agentAddonOptions: agentAddonOptions}
}

func (a *DefaultAgentAddon) Manifests(
	cluster *clusterv1.ManagedCluster,
	addon *addonapiv1alpha1.ManagedClusterAddOn) ([]runtime.Object, error) {
	var objects []runtime.Object

	manifestConfig, err := a.getManifestConfig(cluster, addon)
	if err != nil {
		return objects, err
	}
	for file, data := range a.templateData {
		raw := assets.MustCreateAssetFromTemplate(file, data, manifestConfig).Data
		object, _, err := a.decoder.Decode(raw, nil, nil)
		if err != nil {
			return nil, err
		}
		objects = append(objects, object)
	}

	return objects, nil
}

func (a *DefaultAgentAddon) GetAgentAddonOptions() agent.AgentAddonOptions {
	return a.agentAddonOptions
}

func (a *DefaultAgentAddon) getManifestConfig(
	cluster *clusterv1.ManagedCluster,
	addon *addonapiv1alpha1.ManagedClusterAddOn) (map[string]interface{}, error) {
	manifestConfig := map[string]interface{}{}
	userConfig := a.templateConfig

	typ := reflect.TypeOf(userConfig)
	val := reflect.ValueOf(userConfig)
	for i := 0; i < typ.NumField(); i++ {
		tf := typ.Field(i)
		vf := val.Field(i)
		manifestConfig[tf.Name] = vf.Interface()
	}

	manifestConfig[clusterName] = cluster.GetName()

	installNamespace := addon.Spec.InstallNamespace
	if len(installNamespace) == 0 {
		installNamespace = AddonDefaultInstallNamespace
	}
	manifestConfig[addonInstallNamespace] = installNamespace
	manifestConfig[kubeConfigSecret] = fmt.Sprintf("%s-hub-kubeconfig", a.agentAddonOptions.AddonName)

	// users can override the template config fields by the annotation of addon cr.
	// the key of the annotation is `templateConfig`, the value is a json string which has the config struct.
	// for example: "templateConfig": `{"NodeSelector":{"host":"ssd"},"Image":"quay.io/helloworld:2.4"}`
	annotations := addon.GetAnnotations()
	if len(annotations[templateConfigName]) == 0 {
		return manifestConfig, nil
	}
	annotationConfig := map[string]interface{}{}
	err := json.Unmarshal([]byte(annotations[templateConfigName]), &annotationConfig)
	if err != nil {
		klog.Error("failed to unmarshal the template config from annotation of addon cr. err:%v", err)
		return manifestConfig, err
	}

	for k, v := range annotationConfig {
		manifestConfig[k] = v
	}

	return manifestConfig, nil
}

// validateTemplateData validate template render and object decoder using  an empty cluster and addon
func (a *DefaultAgentAddon) validateTemplateData(file string, data []byte) error {
	config, err := a.getManifestConfig(&clusterv1.ManagedCluster{}, &addonapiv1alpha1.ManagedClusterAddOn{})
	if err != nil {
		return err
	}
	raw := assets.MustCreateAssetFromTemplate(file, data, config).Data
	_, _, err = a.decoder.Decode(raw, nil, nil)
	return err
}

func (a *DefaultAgentAddon) addTemplateData(file string, data []byte) {
	if a.templateData == nil {
		a.templateData = map[string][]byte{}
	}
	a.templateData[file] = data
}

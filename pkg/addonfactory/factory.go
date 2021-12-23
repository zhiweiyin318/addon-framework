package addonfactory

import (
	"embed"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
	"open-cluster-management.io/addon-framework/pkg/agent"
)

const AddonDefaultInstallNamespace = "open-cluster-management-agent-addon"

// AgentAddonFactory builds an agentAddon instance from template files and config .
type AgentAddonFactory struct {
	scheme            *runtime.Scheme
	fs                embed.FS
	templateFiles     []string
	templateConfig    interface{}
	agentAddonOptions agent.AgentAddonOptions
}

func NewAgentAddonFactoryWithTemplates(addonName string, fs embed.FS, templateFiles ...string) *AgentAddonFactory {
	return &AgentAddonFactory{
		fs:            fs,
		templateFiles: templateFiles,
		agentAddonOptions: agent.AgentAddonOptions{
			AddonName:       addonName,
			Registration:    nil,
			InstallStrategy: nil,
		},
	}
}

// WithScheme is an optional configuration, only used when the manifests have customized resource types.
func (f *AgentAddonFactory) WithScheme(scheme *runtime.Scheme) *AgentAddonFactory {
	f.scheme = scheme
	return f
}

// WithTemplateConfig adds the template config struct.
// the reserved fields (ClusterName,AddonInstallNamespace and KubeConfigSecret) are not mandatory in the config.
func (f *AgentAddonFactory) WithTemplateConfig(config interface{}) *AgentAddonFactory {
	f.templateConfig = config
	return f
}

// WithInstallStrategy defines the installation strategy of the manifests prescribed by Manifests(..).
func (f *AgentAddonFactory) WithInstallStrategy(strategy *agent.InstallStrategy) *AgentAddonFactory {
	if strategy.InstallNamespace == "" {
		strategy.InstallNamespace = AddonDefaultInstallNamespace
	}
	f.agentAddonOptions.InstallStrategy = strategy

	return f
}

// WithAgentRegistrationOption defines how agent is registered to the hub cluster.
func (f *AgentAddonFactory) WithAgentRegistrationOption(option *agent.RegistrationOption) *AgentAddonFactory {
	f.agentAddonOptions.Registration = option
	return f
}

// Build creates a new agentAddon.
func (f *AgentAddonFactory) Build() (agent.AgentAddon, error) {
	if len(f.templateFiles) == 0 {
		return nil, fmt.Errorf("there is no template files")
	}

	if f.scheme == nil {
		f.scheme = runtime.NewScheme()
		_ = scheme.AddToScheme(f.scheme)
	}

	agentAddon := newDefaultAgentAddon(f.scheme, f.templateConfig, f.agentAddonOptions)
	for _, file := range f.templateFiles {
		template, err := f.fs.ReadFile(file)
		if err != nil {
			return nil, err
		}

		if err := agentAddon.validateTemplateData(file, template); err != nil {
			return nil, err
		}

		agentAddon.addTemplateData(file, template)
	}

	return agentAddon, nil
}

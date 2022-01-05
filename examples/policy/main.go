package main

import (
	"context"
	"embed"
	"fmt"
	"math/rand"
	"os"
	"time"

	goflag "flag"

	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	utilflag "k8s.io/component-base/cli/flag"
	"k8s.io/component-base/logs"
	"open-cluster-management.io/addon-framework/pkg/addonfactory"
	"open-cluster-management.io/addon-framework/pkg/addonmanager"
	"open-cluster-management.io/addon-framework/pkg/agent"
	"open-cluster-management.io/addon-framework/pkg/utils"
	"open-cluster-management.io/addon-framework/pkg/version"
)

const addonName = "klusterlet-addon-policy"

//go:embed manifests
var fs embed.FS

var manifestFiles = []string{
	"manifests/service_account.yaml",
	"manifests/cluster_role.yaml",
	"manifests/cluster_role_binding.yaml",
	"manifests/config_policy_deployment.yaml",
	"manifests/policy_framework_deployment.yaml",
	"manifests/policy.open-cluster-management.io_configurationpolicies_crd.yaml",
	"manifests/policy.open-cluster-management.io_policies_crd.yaml",
}

type templateConfig struct {
	// Reserved fields
	ClusterName           string
	AddonInstallNamespace string
	KubeConfigSecret      string
	// Custom fields
	IsHubCluster                    bool
	IsCRDV1beta1                    bool
	AddonName                       string
	HTTPSProxy                      string
	HTTPProxy                       string
	NOProxy                         string
	NodeSelector                    map[string]string
	ImagePullSecret                 string
	PolicyControllerImage           string
	GovernancePolicySpecSyncImage   string
	GovernancePolicyStatusSyncImage string
}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())

	pflag.CommandLine.SetNormalizeFunc(utilflag.WordSepNormalizeFunc)
	pflag.CommandLine.AddGoFlagSet(goflag.CommandLine)

	logs.InitLogs()
	defer logs.FlushLogs()

	command := newCommand()
	if err := command.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func newCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "addon",
		Short: "policy addon",
		Run: func(cmd *cobra.Command, args []string) {
			if err := cmd.Help(); err != nil {
				fmt.Fprintf(os.Stderr, "%v\n", err)
			}
			os.Exit(1)
		},
	}

	if v := version.Get().String(); len(v) == 0 {
		cmd.Version = "<unknown>"
	} else {
		cmd.Version = v
	}

	cmd.AddCommand(newControllerCommand())

	return cmd
}

func newControllerCommand() *cobra.Command {
	cmd := controllercmd.
		NewControllerCommandConfig("addon-controller", version.Get(), runController).
		NewCommand()
	cmd.Use = "controller"
	cmd.Short = "Start the addon controller"

	return cmd
}

func runController(ctx context.Context, controllerContext *controllercmd.ControllerContext) error {
	mgr, err := addonmanager.New(controllerContext.KubeConfig)
	if err != nil {
		return err
	}

	agentAddon, err := addonfactory.NewAgentAddonFactoryWithTemplates(addonName, fs, manifestFiles...).
		WithTemplateConfig(newTemplateConfig()).
		WithAgentRegistrationOption(newRegistrationOption()).
		Build()
	if err != nil {
		return err
	}

	mgr.AddAgent(agentAddon)
	mgr.Start(ctx)

	<-ctx.Done()

	return nil
}

func newTemplateConfig() templateConfig {
	return templateConfig{
		AddonName:                       addonName,
		ImagePullSecret:                 "open-cluster-management-image-pull-credentials",
		PolicyControllerImage:           "quay.io/open-cluster-management/config-policy-controller:latest-dev",
		GovernancePolicySpecSyncImage:   "quay.io/open-cluster-management/governance-policy-spec-sync:latest-dev",
		GovernancePolicyStatusSyncImage: "quay.io/open-cluster-management/governance-policy-status-sync:latest-dev",
	}
}

func newRegistrationOption() *agent.RegistrationOption {
	return &agent.RegistrationOption{
		CSRConfigurations: agent.KubeClientSignerConfigurations(addonName, addonName),
		CSRApproveCheck:   utils.DefaultCSRApprover(addonName),
	}
}

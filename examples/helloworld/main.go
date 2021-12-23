package main

import (
	"context"
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
	"open-cluster-management.io/addon-framework/pkg/agent"
	"open-cluster-management.io/addon-framework/pkg/version"

	"open-cluster-management.io/addon-framework/pkg/addonmanager"
)

// addOnAgentInstallationNamespace is the namespace on the managed cluster to install the helloworld addon agent.
const addOnAgentInstallationNamespace = "default"

const addonName = "helloworld"

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
		Short: "helloworld example addon",
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
	cmd.AddCommand(newAgentCommand())

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
		WithInstallStrategy(&agent.InstallStrategy{Type: agent.InstallAll, InstallNamespace: addOnAgentInstallationNamespace}).
		WithAgentRegistrationOption(newRegistrationOption(controllerContext.KubeConfig, controllerContext.EventRecorder, addonName)).
		Build()
	if err != nil {
		return err
	}

	mgr.AddAgent(agentAddon)
	mgr.Start(ctx)

	<-ctx.Done()

	return nil
}

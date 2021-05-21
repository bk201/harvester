package cmd

import (
	"context"
	"fmt"

	"github.com/rancher/wrangler/pkg/signals"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	apiserver "github.com/harvester/harvester/pkg/server"
	"github.com/harvester/harvester/pkg/webhook/config"
	"github.com/harvester/harvester/pkg/webhook/server"
)

var webhookOptions config.Options

var CmdWebhook = cli.Command{
	Name:  "webhook",
	Usage: "start the admission webhook server",
	Flags: []cli.Flag{
		cli.IntFlag{
			Name:        "threadiness",
			EnvVar:      "THREADINESS",
			Usage:       "Specify controller threads",
			Value:       5,
			Destination: &webhookOptions.Threadiness,
		},
		cli.IntFlag{
			Name:        "https-port",
			EnvVar:      "HARVESTER_WEBHOOK_SERVER_HTTPS_PORT",
			Usage:       "HTTPS listen port",
			Value:       9443,
			Destination: &webhookOptions.HTTPSListenPort,
		},
		cli.StringFlag{
			Name:        "namespace",
			EnvVar:      "NAMESPACE",
			Destination: &webhookOptions.Namespace,
			Usage:       "The harvester namespace",
			Required:    true,
		},
	},
	Action: func(c *cli.Context) error {
		return webhookRun()
	},
}

func webhookRun() error {
	logrus.Info("Starting webhook server")

	ctx := signals.SetupSignalHandler(context.Background())

	kubeConfig, err := apiserver.GetConfig(globalOptions.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to find kubeconfig: %v", err)
	}

	restCfg, err := kubeConfig.ClientConfig()
	if err != nil {
		return err
	}

	webhookOptions.HarvesterControllerUsername = fmt.Sprintf("system:serviceaccount:%s:harvester", webhookOptions.Namespace)

	s := server.New(ctx, restCfg, webhookOptions)
	if err := s.ListenAndServe(); err != nil {
		return err
	}
	<-ctx.Done()
	return nil
}

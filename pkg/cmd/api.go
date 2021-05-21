package cmd

import (
	"context"
	"fmt"

	"github.com/rancher/wrangler/pkg/signals"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/harvester/harvester/pkg/config"
	"github.com/harvester/harvester/pkg/server"
)

var apiOptions config.Options

var CmdAPI = cli.Command{
	Name:  "api",
	Usage: "start the API server",
	Flags: []cli.Flag{
		cli.IntFlag{
			Name:        "threadiness",
			EnvVar:      "THREADINESS",
			Usage:       "Specify controller threads",
			Value:       10,
			Destination: &apiOptions.Threadiness,
		},
		cli.IntFlag{
			Name:        "http-port",
			EnvVar:      "HARVESTER_SERVER_HTTP_PORT",
			Usage:       "HTTP listen port",
			Value:       8080,
			Destination: &apiOptions.HTTPListenPort,
		},
		cli.IntFlag{
			Name:        "https-port",
			EnvVar:      "HARVESTER_SERVER_HTTPS_PORT",
			Usage:       "HTTPS listen port",
			Value:       8443,
			Destination: &apiOptions.HTTPSListenPort,
		},
		cli.StringFlag{
			Name:        "namespace",
			EnvVar:      "NAMESPACE",
			Destination: &apiOptions.Namespace,
			Usage:       "The default namespace to store management resources",
			Required:    true,
		},
		cli.BoolFlag{
			Name:        "skip-authentication",
			EnvVar:      "SKIP_AUTHENTICATION",
			Usage:       "Define whether to skip auth login or not, default to false",
			Destination: &apiOptions.SkipAuthentication,
		},
		cli.StringFlag{
			Name:   "authentication-mode",
			EnvVar: "HARVESTER_AUTHENTICATION_MODE",
			Usage:  "Define authentication mode, kubernetesCredentials, localUser and rancher are supported, could config more than one mode, separated by comma",
		},
		cli.BoolFlag{
			Name:        "hci-mode",
			EnvVar:      "HCI_MODE",
			Usage:       "Enable HCI mode. Additional controllers are registered in HCI mode",
			Destination: &apiOptions.HCIMode,
		},
		cli.BoolFlag{
			Name:        "rancher-embedded",
			EnvVar:      "RANCHER_EMBEDDED",
			Usage:       "Specify whether the Harvester is running with embedded Rancher mode, default to false",
			Destination: &apiOptions.RancherEmbedded,
		},
		cli.StringFlag{
			Name:        "rancher-server-url",
			EnvVar:      "RANCHER_SERVER_URL",
			Usage:       "Specify the URL to connect to the Rancher server",
			Destination: &apiOptions.RancherURL,
		},
	},
	Action: func(c *cli.Context) error {
		return apiRun(c, apiOptions)
	},
}

func apiRun(c *cli.Context, options config.Options) error {
	logrus.Info("Starting controller")
	ctx := signals.SetupSignalHandler(context.Background())

	kubeConfig, err := server.GetConfig(globalOptions.KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to find kubeconfig: %v", err)
	}

	harv, err := server.New(ctx, kubeConfig, options)
	if err != nil {
		return fmt.Errorf("failed to create harvester server: %v", err)
	}
	return harv.ListenAndServe(nil, options)
}

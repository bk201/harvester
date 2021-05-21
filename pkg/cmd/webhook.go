package cmd

import (
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"
)

var CmdWebhook = cli.Command{
	Name:  "webhook",
	Usage: "start the admission webhook server",
	Action: func(c *cli.Context) error {
		return webhookRun()
	},
}

func webhookRun() error {
	logrus.Info("Starting webhook server")
	return nil
}

package cmd

import (
	"log"
	"net/http"
	"os"

	"github.com/ehazlett/simplelog"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli"

	"github.com/harvester/harvester/pkg/config"
	"github.com/harvester/harvester/pkg/version"
)

var globalOptions config.GlobalOptions

func Execute() {
	app := cli.NewApp()
	app.Name = "harvester"
	app.Version = version.FriendlyVersion()
	app.Usage = ""

	app.Commands = []cli.Command{
		CmdAPI,
		CmdWebhook,
	}

	// global flags
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "kubeconfig",
			EnvVar:      "KUBECONFIG",
			Usage:       "Kube config for accessing k8s cluster",
			Destination: &globalOptions.KubeConfig,
		},
		cli.StringFlag{
			Name:        "profile-listen-address",
			Value:       "0.0.0.0:6060",
			Usage:       "Address to listen on for profiling",
			Destination: &globalOptions.ProfilerAddress,
		},
		cli.BoolFlag{
			Name:        "debug",
			EnvVar:      "HARVESTER_DEBUG",
			Usage:       "Enable debug logs",
			Destination: &globalOptions.Debug,
		},
		cli.BoolFlag{
			Name:        "trace",
			EnvVar:      "HARVESTER_TRACE",
			Usage:       "Enable trace logs",
			Destination: &globalOptions.Trace,
		},
		cli.StringFlag{
			Name:        "log-format",
			EnvVar:      "HARVESTER_LOG_FORMAT",
			Usage:       "Log format",
			Value:       "text",
			Destination: &globalOptions.LogFormat,
		},
	}

	app.Before = func(c *cli.Context) error {
		initProfiling(globalOptions)
		initLogs(globalOptions)
		return nil
	}

	if err := app.Run(os.Args); err != nil {
		logrus.Fatal(err)
	}
}

func initProfiling(options config.GlobalOptions) {
	// enable profiler
	if options.ProfilerAddress != "" {
		go func() {
			log.Println(http.ListenAndServe(options.ProfilerAddress, nil))
		}()
	}
}

func initLogs(options config.GlobalOptions) {
	switch options.LogFormat {
	case "simple":
		logrus.SetFormatter(&simplelog.StandardFormatter{})
	case "json":
		logrus.SetFormatter(&logrus.JSONFormatter{})
	default:
		logrus.SetFormatter(&logrus.TextFormatter{})
	}
	logrus.SetOutput(os.Stdout)
	if options.Debug {
		logrus.SetLevel(logrus.DebugLevel)
		logrus.Debugf("Loglevel set to [%v]", logrus.DebugLevel)
	}
	if options.Trace {
		logrus.SetLevel(logrus.TraceLevel)
		logrus.Tracef("Loglevel set to [%v]", logrus.TraceLevel)
	}
}

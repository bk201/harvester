package config

type GlobalOptions struct {
	Debug     bool
	Trace     bool
	LogFormat string

	ProfilerAddress string
	KubeConfig      string
}

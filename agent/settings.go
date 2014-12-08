package agent

import (
	"flag"
	"os"
)

type AgentSettings struct {
	AgentConfigFile     string
	Verbose             bool
	SentinelIp          string `yaml:"sentinel_ip"`
	SentinelPort        string `yaml:"sentinel_port"`
	RestartCommand      string `yaml:"restart_command"`
	LogFile             string `yaml:"log_file"`
	AppConfig           string `yaml:"app_config"`
}

var Settings AgentSettings = AgentSettings{}

func ValidateSettings() {
	if
		Settings.SentinelIp     == "" || 
		Settings.SentinelPort   == "" || 
		Settings.AppConfig      == "" {
		flag.Usage()
		os.Exit(1)
	}
}

func init() {
	flag.StringVar(&Settings.AgentConfigFile, "c", "conf/agent.yml", "set configuration file")
	flag.BoolVar(&Settings.Verbose, "verbose", false, "Log generic info")
	flag.Parse()
	ReadYaml(Settings.AgentConfigFile, &Settings)

	SetFileLogger()
	ValidateSettings()
}

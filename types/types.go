package types

// Config is a struct of the framework configuration
type Config struct {
	LogLevel           string
	MinVersion         string
	AppName            string
	EnableSyslog       bool
	MesosAgentHostname string
	FrameworkID        string
	AgentID            string
	ExecutorID         string
	SkipSSL            bool
}

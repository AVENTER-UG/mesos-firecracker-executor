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
	ExecutorHostname   string
	ExecutorPort       string
	SkipSSL            bool
	SSLKey             string
	SSLCrt             string
	Listen             string
}

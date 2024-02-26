package main

import (
	"os"
	"sort"
	"strings"

	"github.com/AVENTER-UG/mesos-firecracker-executor/mesos"
	"github.com/AVENTER-UG/mesos-firecracker-executor/mesosdriver"
	"github.com/AVENTER-UG/util/util"
	mesosconfig "github.com/mesos/mesos-go/api/v1/lib/executor/config"
	"github.com/sirupsen/logrus"
)

// BuildVersion of m3s
var BuildVersion string

// GitVersion is the revision and commit number
var GitVersion string

// Settings of the firecracker executor
var Settings map[string]string

func logConfig() {
	logrus.WithField("func", "main.logConfig").Info("Environment ---------------------------")
	envVars := os.Environ()
	sort.Strings(envVars)
	Settings = make(map[string]string)
	for _, setting := range envVars {

		if strings.HasPrefix(setting, "MESOS") ||
			strings.HasPrefix(setting, "FIRECRACKER") ||
			(setting == "HOME") {

			pair := strings.Split(setting, "=")
			logrus.Infof(" * %-30s: %s", pair[0], pair[1])
			Settings[pair[0]] = pair[1]
		}
	}
	logrus.WithField("func", "main.logConfig").Info("---------------------------------------")
}

func main() {

	logrus.WithField("func", "main").Println("mesos-firecracker-executor build " + BuildVersion + " git " + GitVersion)

	logConfig()

	Settings["FIRECRACKER_VCPU"] = util.Getenv("FIRECRACKER_VCPU", "1")
	Settings["FIRECRACKER_MEM_MB"] = util.Getenv("FIRECRACKER_MEM_MB", "256")
	Settings["FIRECRACKER_AGENT_PORT"] = util.Getenv("FIRECRACKER_AGENT_PORT", "8085")
	Settings["FIRECRACKER_WORKDIR"] = util.Getenv("FIRECRACKER_WORKDIR", "/mnt/mesos/sandbox")
	Settings["FIRECRACKER_PAYLOAD_FILE"] = util.Getenv("FIRECRACKER_PAYLOAD_FILE", "8085")

	cfg, err := mesosconfig.FromEnv()
	if err != nil {
		logrus.WithField("func", "main").Fatal("failed to load configuration: " + err.Error())
	}

	var level logrus.Level
	level, err = logrus.ParseLevel("TRACE")
	if err != nil {
		return
	}
	logrus.SetLevel(level)

	nExec := mesos.NewExecutor(&cfg, Settings)
	nExec.Driver = mesosdriver.NewExecutorDriver(&cfg, nExec)
	err = nExec.Driver.Run()
	if err != nil {
		logrus.WithField("func", "main").Errorf("Immediate Exit: Error from executor driver: %s", err)
		return
	}

	logrus.WithField("func", "main").Info("Sidecar Executor exiting")

}

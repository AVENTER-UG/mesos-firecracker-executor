package mesos

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"net/http"
	"strings"
	"time"

	"github.com/gogo/protobuf/jsonpb"
	"github.com/sirupsen/logrus"

	"github.com/AVENTER-UG/mesos-firecracker-executor/executor"
	cfg "github.com/AVENTER-UG/mesos-firecracker-executor/types"
	mesosutil "github.com/AVENTER-UG/mesos-util"
	mesosproto "github.com/AVENTER-UG/mesos-util/proto"
)

type MExecutor cfg.Executor

// Marshaler to serialize Protobuf Message to JSON
var marshaller = jsonpb.Marshaler{
	EnumsAsInts: false,
	Indent:      " ",
	OrigName:    true,
}

var config *cfg.Config
var framework *mesosutil.FrameworkConfig
var e *cfg.Executor

// SetConfig set the global config
func SetConfig(cfg *cfg.Config, frm *mesosutil.FrameworkConfig) {
	config = cfg
	framework = frm
}

// NewExecutor initializes a new executor with the given executor and framework ID
func Executor() error {
	protocol := "https"
	if !framework.MesosSSL {
		protocol = "http"
	}

	subscribeCall := &executor.Call{
		Type: executor.Call_SUBSCRIBE,
		FrameworkID: mesosproto.FrameworkID{
			Value: config.FrameworkID,
		},
		ExecutorID: mesosproto.ExecutorID{
			Value: config.ExecutorID,
		},
		Subscribe: &executor.Call_Subscribe{
			UnacknowledgedTasks:   []mesosproto.TaskInfo{},
			UnacknowledgedUpdates: []executor.Call_Update{},
		},
	}

	body, _ := marshaller.MarshalToString(subscribeCall)
	logrus.Debug(body)
	client := &http.Client{}
	client.Transport = &http.Transport{
		// #nosec G402
		ResponseHeaderTimeout: 30 * time.Second,
		MaxIdleConnsPerHost:   2,
		TLSClientConfig:       &tls.Config{InsecureSkipVerify: config.SkipSSL},
	}

	req, _ := http.NewRequest("POST", protocol+"://"+config.MesosAgentHostname+"/api/v1/executor", bytes.NewBuffer([]byte(body)))
	req.Close = true
	req.SetBasicAuth(framework.Username, framework.Password)
	req.Header.Set("Content-Type", "application/json")

	// Prepare executor
	e = &cfg.Executor{
		Cli:           client,
		Req:           req,
		ExecutorID:    config.ExecutorID,
		FrameworkID:   config.FrameworkID,
		FrameworkInfo: mesosproto.FrameworkInfo{},
	}

	return Subscribe()
}

// Execute runs the executor workflow
func Subscribe() error {
	var err error
	for !e.Shutdown {
		var res *http.Response
		res, err = e.Cli.Do(e.Req)
		if res != nil {
			defer res.Body.Close()
		}
		if err != nil {
			logrus.Error("Error during connect: ", err.Error())
			return err
		}
		reader := bufio.NewReader(res.Body)
		line, _ := reader.ReadString('\n')
		line = strings.TrimSuffix(line, "\n")

		// We are connected, we start to handle events
		for !e.Shutdown {
			line, _ := reader.ReadString('\n')
			line = strings.TrimSuffix(line, "\n")
			var event executor.Event
			err = jsonpb.UnmarshalString(line, &event)
			if err != nil {
				logrus.Error("Error during unmarshal: ", err.Error())
				return err
			}
			logrus.Debug("Subscribe Got: ", event.GetType())

			switch event.Type {
			case executor.Event_SUBSCRIBED:
				err = handleSubscribed(&event)

			case executor.Event_LAUNCH_GROUP:
				err = handleLaunchGroup(&event)

			case executor.Event_LAUNCH:
				err = handleLaunch(&event)

			default:
				logrus.Debug("DEFAULT EVENT: ", event.Type)
			}
		}
	}

	// Now, executor is shutting down (every tasks have been killed)
	logrus.Info("All tasks have been killed. Now exiting, bye bye.")

	return err
}

// handleSubscribed handles subscribed events
func handleSubscribed(ev *executor.Event) error {
	logrus.Info("Handled SUBSCRIBED event")
	logrus.Debug(ev)

	e.AgentInfo = ev.GetSubscribed().GetAgentInfo()
	e.ExecutorInfo = ev.GetSubscribed().GetExecutorInfo()
	e.FrameworkInfo = ev.GetSubscribed().GetFrameworkInfo()

	return nil
}

// handleLaunchGroup puts given task in unacked tasks and launches it
func handleLaunchGroup(ev *executor.Event) error {
	logrus.Info("Handled LAUNCH Group event")
	logrus.Debug(ev)
	e.TaskInfo = ev.GetLaunch().GetTask()

	return nil
}

// handleLaunch puts given task in unacked tasks and launches it
func handleLaunch(ev *executor.Event) error {
	logrus.Info("Handled LAUNCH event")
	logrus.Debug(ev)
	e.TaskInfo = ev.GetLaunch().GetTask()

	logrus.Debug(e.TaskInfo)

	return nil
}

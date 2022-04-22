package mesos

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"net/http"
	"strings"

	executor "github.com/AVENTER-UG/mesos-mainframe-executor/executor"
	cfg "github.com/AVENTER-UG/mesos-mainframe-executor/types"
	mesosutil "github.com/AVENTER-UG/mesos-util"
	mesosproto "github.com/AVENTER-UG/mesos-util/proto"
	"github.com/sirupsen/logrus"

	"github.com/gogo/protobuf/jsonpb"
)

// Service include all the current vars and global config
var config *cfg.Config
var framework *mesosutil.FrameworkConfig

// Marshaler to serialize Protobuf Message to JSON
var marshaller = jsonpb.Marshaler{
	EnumsAsInts: false,
	Indent:      " ",
	OrigName:    true,
}

// SetConfig set the global config
func SetConfig(cfg *cfg.Config, frm *mesosutil.FrameworkConfig) {
	config = cfg
	framework = frm
}

// Subscribe to the mesos backend
func Subscribe() error {
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
	logrus.Debug(subscribeCall)
	body, _ := marshaller.MarshalToString(subscribeCall)
	logrus.Debug(body)
	client := &http.Client{}
	client.Transport = &http.Transport{
		// #nosec G402
		TLSClientConfig: &tls.Config{InsecureSkipVerify: config.SkipSSL},
	}

	protocol := "https"
	if !framework.MesosSSL {
		protocol = "http"
	}
	req, _ := http.NewRequest("POST", protocol+"://"+config.MesosAgentHostname+"/api/v1/executor", bytes.NewBuffer([]byte(body)))
	req.Close = true
	req.SetBasicAuth(framework.Username, framework.Password)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)

	if err != nil {
		logrus.Fatal("Error during subscribe: ", err.Error())
		return err
	}
	defer res.Body.Close()

	reader := bufio.NewReader(res.Body)

	line, _ := reader.ReadString('\n')
	line = strings.TrimSuffix(line, "\n")

	for {
		// Read line from Mesos
		line, _ = reader.ReadString('\n')
		line = strings.TrimSuffix(line, "\n")
		var event executor.Event
		err := jsonpb.UnmarshalString(line, &event)
		if err != nil {
			logrus.Error("Error during unmarshal: ", err.Error())
			return err
		}
		logrus.Debug("Subscribe Got: ", event.GetType())

		switch event.Type {
		case executor.Event_SUBSCRIBED:
			logrus.Debug(event)
			logrus.Info("Subscribed")
			logrus.Info("ExecutorId: ", event.Subscribed.ExecutorInfo.ExecutorID)

		case executor.Event_MESSAGE:
			logrus.Debug(event)
			logrus.Info("Message")
		case executor.Event_LAUNCH_GROUP:
			logrus.Debug(event)
			logrus.Info("Launch Group")

		case executor.Event_LAUNCH:
			logrus.Debug(event)
			logrus.Info("Launch")
		case executor.Event_ACKNOWLEDGED:
			logrus.Debug(event)
			logrus.Info("Acknowledged")
		case executor.Event_ERROR:
			logrus.Debug(event)
			logrus.Info("Error")
		case executor.Event_KILL:
			logrus.Debug(event)
			logrus.Info("Kill")
		case executor.Event_SHUTDOWN:
			logrus.Debug(event)
			logrus.Info("Shutdown")
		case executor.Event_UNKNOWN:
			logrus.Debug(event)
			logrus.Info("Unknown")
		case executor.Event_HEARTBEAT:
			logrus.Debug(event)
			logrus.Info("Heartbeat")

		default:
			logrus.Debug("DEFAULT EVENT: ", event.Type)
		}
	}
}

// Call will send messages to mesos
func Call(message *executor.Call) {
	body, _ := marshaller.MarshalToString(message)

	client := &http.Client{}
	client.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	protocol := "https"
	if !framework.MesosSSL {
		protocol = "http"
	}
	req, _ := http.NewRequest("POST", protocol+"://"+config.MesosAgentHostname+"/api/v1/executor", bytes.NewBuffer([]byte(body)))
	req.Close = true
	req.SetBasicAuth(framework.Username, framework.Password)
	req.Header.Set("Content-Type", "application/json")
	res, err := client.Do(req)

	if err != nil {
		logrus.Debug("Call Message: ", err)
	}

	defer res.Body.Close()
}

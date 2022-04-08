package mesos

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"net/http"
	"strconv"
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
		Subscribe: &executor.Call_Subscribe{},
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
	bytesCount, _ := strconv.Atoi(strings.Trim(line, "\n"))

	for {
		// Read line from Mesos
		line, _ = reader.ReadString('\n')
		line = strings.Trim(line, "\n")
		// Read important data
		data := line[:bytesCount]
		// Rest data will be bytes of next message
		bytesCount, _ = strconv.Atoi((line[bytesCount:]))
		var event executor.Event
		err := jsonpb.UnmarshalString(data, &event)
		if err != nil {
			logrus.Error("Error during unmarshal: ", err.Error())
			return err
		}
		logrus.Debug("Subscribe Got: ", event.GetType())

		switch event.Type {
		case executor.Event_SUBSCRIBED:
			logrus.Debug(event)
			logrus.Info("Subscribed")

		default:
			logrus.Debug("DEFAULT EVENT: ", event.Type)
		}
	}
}
package router

import (
	"fmt"
	"strings"

	"github.com/futurehomeno/fimpgo"
	"github.com/maverich/corona-ad/model"
	log "github.com/sirupsen/logrus"
)

type FromFimpRouter struct {
	inboundMsgCh fimpgo.MessageCh
	mqt          *fimpgo.MqttTransport
	instanceId   string
	appLifecycle *model.Lifecycle
	configs      *model.Configs
}

func NewFromFimpRouter(mqt *fimpgo.MqttTransport, appLifecycle *model.Lifecycle, configs *model.Configs) *FromFimpRouter {
	fc := FromFimpRouter{inboundMsgCh: make(fimpgo.MessageCh, 5), mqt: mqt, appLifecycle: appLifecycle, configs: configs}
	fc.mqt.RegisterChannel("ch1", fc.inboundMsgCh)
	return &fc
}

func (fc *FromFimpRouter) Start() {
	//TODO: Choose either adapter or app topic

	// Adapter topics
	fc.mqt.Subscribe(fmt.Sprintf("pt:j1/+/rt:dev/rn:%s/ad:1/#", model.ServiceName))
	fc.mqt.Subscribe(fmt.Sprintf("pt:j1/+/rt:ad/rn:%s/ad:1", model.ServiceName))

	// App topics
	// fc.mqt.Subscribe(fmt.Sprintf("pt:j1/+/rt:app/rn:%s/ad:1",ServiceName))

	go func(msgChan fimpgo.MessageCh) {
		for {
			select {
			case newMsg := <-msgChan:
				fc.routeFimpMessage(newMsg)

			}
		}

	}(fc.inboundMsgCh)
}

func (fc *FromFimpRouter) routeFimpMessage(newMsg *fimpgo.Message) {
	log.Debug("New fimp msg")
	addr := strings.Replace(newMsg.Addr.ServiceAddress, "_0", "", 1)
	switch newMsg.Payload.Service {
	case "out_lvl_switch":
		addr = strings.Replace(addr, "l", "", 1)
		switch newMsg.Payload.Type {
		case "cmd.binary.set":
			//val,_ := newMsg.Payload.GetBoolValue()
			// TODO: Add your logic here

			//log.Debug("Status code = ",respH.StatusCode)
		case "cmd.lvl.set":
			//val,_ := newMsg.Payload.GetIntValue()
			// TODO: Add your logic here
		}

	case "out_bin_switch":
		log.Debug("Sending switch")
		//val,_ := newMsg.Payload.GetBoolValue()
		// TODO: Add your logic here
	case model.ServiceName:
		adr := &fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeAdapter, ResourceName: model.ServiceName, ResourceAddress: "1"}
		switch newMsg.Payload.Type {
		case "cmd.auth.login":
			reqVal, err := newMsg.Payload.GetStrMapValue()
			status := "ok"
			if err != nil {
				log.Error("Incorrect login message ")
				return
			}
			username, _ := reqVal["username"]
			password, _ := reqVal["password"]
			if username != "" && password != "" {
				// TODO: Add your logic here
			}
			fc.configs.SaveToFile()
			if err != nil {
				status = "error"
			}
			msg := fimpgo.NewStringMessage("evt.system.login_report", model.ServiceName, status, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}
		case "cmd.system.get_connect_params":
			val := map[string]string{"host": "", "username": "", "password": ""}
			msg := fimpgo.NewStrMapMessage("evt.system.connect_params_report", model.ServiceName, val, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.config.set":
			configs, err := newMsg.Payload.GetStrMapValue()
			if err != nil {
				return
			}
			log.Debugf("App reconfigured . New parameters : %v", configs)
			//TODO: Add your logic here

		case "cmd.log.set_level":
			level, err := newMsg.Payload.GetStringValue()
			if err != nil {
				return
			}
			logLevel, err := log.ParseLevel(level)
			if err == nil {
				log.SetLevel(logLevel)
				fc.configs.LogLevel = level
				fc.configs.SaveToFile()
			}
			log.Info("Log level updated to = ", logLevel)

		case "cmd.system.connect":
			_, err := newMsg.Payload.GetStrMapValue()
			var errStr string
			status := "ok"
			//if err != nil {
			//	log.Error("Incorrect login message ")
			//	errStr = err.Error()
			//}
			//host,_ := reqVal["host"]
			//username,_ := reqVal["username"]
			//password,_ := reqVal["password"]

			//if username != ""{
			fc.appLifecycle.PublishEvent(model.EventConfigured, "from-fimp-router", nil)
			//}
			fc.configs.SaveToFile()
			if err != nil {
				status = "error"
			}
			val := map[string]string{"status": status, "error": errStr}
			msg := fimpgo.NewStrMapMessage("evt.system.connect_report", model.ServiceName, val, nil, nil, newMsg.Payload)
			if err := fc.mqt.RespondToRequest(newMsg.Payload, msg); err != nil {
				fc.mqt.Publish(adr, msg)
			}

		case "cmd.network.get_all_nodes":
			// TODO: Add your logic here
		case "cmd.thing.get_inclusion_report":
			//nodeId , _ := newMsg.Payload.GetStringValue()
			// TODO: Add your logic here
		case "cmd.thing.inclusion":
			//flag , _ := newMsg.Payload.GetBoolValue()
			// TODO: Add your logic here
		case "cmd.thing.delete":
			// remove device from network
			val, err := newMsg.Payload.GetStrMapValue()
			if err != nil {
				log.Error("Wrong msg format")
				return
			}
			deviceId, ok := val["address"]
			if ok {
				// TODO: Add your logic here
				log.Info(deviceId)
			} else {
				log.Error("Incorrect address")

			}

		}
		//

	}

}

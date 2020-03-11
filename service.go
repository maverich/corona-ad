package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/futurehomeno/fimpgo"
	"github.com/futurehomeno/fimpgo/discovery"
	"github.com/maverich/corona-ad/model"
	"github.com/maverich/corona-ad/router"
	log "github.com/sirupsen/logrus"
	"github.com/tidwall/gjson"
	"gopkg.in/natefinch/lumberjack.v2"
)

func SetupLog(logfile string, level string, logFormat string) {
	if logFormat == "json" {
		log.SetFormatter(&log.JSONFormatter{TimestampFormat: "2006-01-02 15:04:05.999"})
	} else {
		log.SetFormatter(&log.TextFormatter{FullTimestamp: true, ForceColors: true, TimestampFormat: "2006-01-02T15:04:05.999"})
	}

	logLevel, err := log.ParseLevel(level)
	if err == nil {
		log.SetLevel(logLevel)
	} else {
		log.SetLevel(log.DebugLevel)
	}

	if logfile != "" {
		l := lumberjack.Logger{
			Filename:   logfile,
			MaxSize:    5, // megabytes
			MaxBackups: 2,
		}
		log.SetOutput(&l)
	}
}

func clientInit() (*http.Client, *http.Request) {
	url := "https://www.vg.no/spesial/2020/corona-viruset/data/norway-table-overview/?region=municipality"
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)

	if err != nil {
		fmt.Println(err)
	}
	req.Header.Add("Host", " www.vg.no")
	req.Header.Add("User-Agent", " Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:73.0) Gecko/20100101 Firefox/73.0")
	req.Header.Add("Accept", " application/json, text/javascript, */*; q=0.01")
	req.Header.Add("Accept-Language", " en-US,en;q=0.5")
	req.Header.Add("Accept-Encoding", " gzip, deflate, br")
	req.Header.Add("X-Requested-With", " XMLHttpRequest")
	req.Header.Add("Connection", " keep-alive")
	req.Header.Add("Referer", " https://www.vg.no/spesial/2020/corona-viruset/enrichment-norway-map.php?initialWidth=690&childId=spesial-2020-corona-viruset-enrichment-norway-map&parentTitle=Kulturminister%20Raja%20om%20coronasituasjonen%3A%20%E2%80%93%20Reell%20fare%20for%20konkurser%20%E2%80%93%20VG&parentUrl=https%3A%2F%2Fwww.vg.no%2Fnyheter%2Fi%2F8mkXJQ%2Fkulturminister-raja-om-coronasituasjonen-reell-fare-for-konkurser")
	req.Header.Add("Cookie", " clientBucket=27; _lp4_u=ADvarpKyXF; _lp4_c=; _pulse2data=146f8f12-8f0e-43ac-83e6-30d09207aadf%2Cv%2C%2C1583926239679%2CeyJpc3N1ZWRBdCI6IjIwMjAtMDEtMjNUMTM6NTY6NTNaIiwiZW5jIjoiQTEyOENCQy1IUzI1NiIsImFsZyI6ImRpciIsImtpZCI6IjIifQ..y-iw1KWsihjLXdpUEBHHVw.Ijs8HXxeYw9--D2k16uNhlDkeoJOc-bM5afqqgwAQX-Vmk3_RlHNtuoT6kp1eJwvPFnpo_xc03XRhqJC7aRx4ACRbi6yR0HJl-fSBdO0w2AAVfaTwFPPb4mS4T7nU3NEzPEQ0IU3RjFWNcO0tFVyZEy6YGrs3goa9QfRUE8Ll-subphHbt3Dq5Nm8vDogyar1M_f_XI8bBXNNWhdPYTs1Q.J8K7Q3O8er47D1MEryzbWA%2C0%2C1583939739679%2Ctrue%2C%2CeyJraWQiOiIyIiwiYWxnIjoiSFMyNTYifQ..vlgptjgQZhoBqRaWs7EFtST-iiqffx9AkTVldtW8KCE")
	return client, req
}

func GetConfirmedNumberOfCases(c *http.Client, req *http.Request) (float64, error) {
	res, _ := c.Do(req)
	defer res.Body.Close()
	body, _ := ioutil.ReadAll(res.Body)
	result := gjson.GetBytes(body, "totals.confirmed").Float()
	return result, nil
}

func main() {
	var configFile string
	flag.StringVar(&configFile, "c", "", "Config file")
	flag.Parse()
	if configFile == "" {
		configFile = "./config.json"
	} else {
		fmt.Println("Loading configs from file ", configFile)
	}
	appLifecycle := model.NewAppLifecycle()
	configs := model.NewConfigs(configFile)
	err := configs.LoadFromFile()
	if err != nil {
		fmt.Print(err)
		panic("Can't load config file.")
	}

	SetupLog(configs.LogFile, configs.LogLevel, configs.LogFormat)
	log.Info("--------------Starting corona-ad----------------")
	appLifecycle.PublishEvent(model.EventConfiguring, "main", nil)

	mqtt := fimpgo.NewMqttTransport(configs.MqttServerURI, configs.MqttClientIdPrefix, configs.MqttUsername, configs.MqttPassword, true, 1, 1)
	err = mqtt.Start()
	if err != nil {
		panic(fmt.Sprintf("Can't connect to broker. Error:", err.Error()))
	}
	defer mqtt.Stop()
	responder := discovery.NewServiceDiscoveryResponder(mqtt)
	responder.RegisterResource(model.GetDiscoveryResource())
	responder.Start()

	fimpRouter := router.NewFromFimpRouter(mqtt, appLifecycle, configs)
	fimpRouter.Start()

	//------------------ Sample code --------------------------------------

	log.Info("Connected")
	client, request := clientInit()
	ticker := time.NewTicker(10 * time.Second)

	for range ticker.C {
		confirmedCases, _ := GetConfirmedNumberOfCases(client, request)
		msg := fimpgo.NewFloatMessage("evt.sensor.report", "sensor_gp", confirmedCases, nil, nil, nil)
		adr := fimpgo.Address{MsgType: fimpgo.MsgTypeEvt, ResourceType: fimpgo.ResourceTypeDevice, ResourceName: "corona-ad", ResourceAddress: "1", ServiceName: "temp_sensor", ServiceAddress: "300"}
		mqtt.Publish(&adr, msg)
	}
}

package api

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/gin-contrib/sse"
	"github.com/ttocsneb/weather-ui/util"
)

type Sensor struct {
	Unit  string
	Value float64
}

type Conditions struct {
	Station string
	Server  string
	Time    time.Time
	Sensors map[string][]Sensor
}

type Info struct {
	Server       string
	Station      string
	Make         string
	Model        string
	Software     string
	Version      string
	Latitude     float64
	Longitude    float64
	Elevation    float64
	District     string
	City         string
	Region       string
	Country      string
	RapidWeather bool
	Updated      time.Time
}

func FetchStationConditions(conf *util.Config, server string, station string) (Conditions, error) {

	cond_url := fmt.Sprintf("%v/station/%v/%v/conditions/", conf.Server, server, station)
	var data Conditions

	content, err := util.FetchDataToBytes(cond_url)
	if err != nil {
		return data, err
	}

	err = json.Unmarshal(content, &data)
	if err != nil {
		return data, err
	}

	return data, nil
}

func FetchStationInfo(conf *util.Config, server string, station string) (Info, error) {
	cond_url := fmt.Sprintf("%v/station/%v/%v/info/", conf.Server, server, station)
	var data Info

	content, err := util.FetchDataToBytes(cond_url)
	if err != nil {
		return data, err
	}

	err = json.Unmarshal(content, &data)
	if err != nil {
		return data, err
	}

	return data, nil
}

var conditionsMux map[string]*util.ChanMultiplex[Conditions]

func fetchConditionsUpdates(url string) (chan Conditions, func()) {
	mux, exists := conditionsMux[url]
	if !exists {
		mux = util.NewChanMultiplex(
			func(cm *util.ChanMultiplex[Conditions], done chan struct{}) {
				fmt.Println("Starting new connection")
				err := util.FetchDataSSE(url, done, func(e sse.Event) {
					data := fmt.Sprint(e.Data)

					var cond Conditions
					err := json.Unmarshal([]byte(data), &cond)
					if err != nil {
						fmt.Printf("Could not unmarshal station condition updates: %v\n", err)
						return
					}

					cm.Notify(cond)
				}, func() {
					fmt.Println("Closing connection")
					cm.Close()
				})
				if err != nil {
					fmt.Printf("Could not Fetch station condition updates: %v\n", err)
				}
			})
		conditionsMux[url] = mux
	}

	event_ch := mux.Subscribe()
	finished := func() {
		mux.Unsubscribe(event_ch)
	}

	return event_ch, finished
}

func FetchStationConditionUpdates(conf *util.Config, server string, station string) (chan Conditions, func()) {
	cond_url := fmt.Sprintf("%v/station/%v/%v/conditions/updates/", conf.Server, server, station)
	return fetchConditionsUpdates(cond_url)
}

func FetchStationRapidConditionUpdates(conf *util.Config, server string, station string) (chan Conditions, func()) {
	cond_url := fmt.Sprintf("%v/station/%v/%v/conditions/rapid/", conf.Server, server, station)
	return fetchConditionsUpdates(cond_url)
}

func setupStation() {
	conditionsMux = make(map[string]*util.ChanMultiplex[Conditions])
	regionConditionsMux = make(map[string]*util.ChanMultiplex[RegionUpdate])
}

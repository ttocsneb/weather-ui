package api

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/gin-contrib/sse"
	"github.com/ttocsneb/weather-ui/util"
)

func FetchRegion(conf *util.Config, country string, region string, city string, district string) (map[string]Sensor, error) {
	args := []string{}
	arg := func(value string) {
		if value != "" {
			args = append(args, util.EncodeURIString(value))
		}
	}

	arg(country)
	arg(region)
	arg(city)
	arg(district)

	params := strings.Join(args, "/")

	url := fmt.Sprintf("%s/region/conditions/%v/", conf.Server, params)

	response, err := util.FetchDataToBytes(url)
	if err != nil {
		return nil, err
	}

	var body map[string]Sensor
	err = json.Unmarshal(response, &body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

type Region struct {
	Country  string
	Region   string
	City     string
	District string
}

func SearchRegion(conf *util.Config, parts ...string) ([]Region, error) {
	args := []string{}
	letter := 'a'
	for _, part := range parts {
		args = append(args, fmt.Sprintf("%c=%v", letter, util.EncodeURIString(part)))
		letter += 1
	}

	params := strings.Join(args, "&")

	url := fmt.Sprintf("%s/region/search/?%v", conf.Server, params)

	response, err := util.FetchDataToBytes(url)
	if err != nil {
		return nil, err
	}

	var result []Region
	err = json.Unmarshal(response, &result)

	return result, err
}

type RegionUpdate map[string]Sensor

var regionConditionsMux map[string]*util.ChanMultiplex[RegionUpdate]

func FetchRegionUpdates(conf *util.Config, country string, region string, city string, district string) (chan RegionUpdate, func()) {
	var url string
	if district == "" {
		url = fmt.Sprintf("%v/region/conditions/updates/%v/%v/%v/", conf.Server, country, region, city)
	} else {
		url = fmt.Sprintf("%v/region/conditions/updates/%v/%v/%v/%v/", conf.Server, country, region, city, district)
	}
	mux, exists := regionConditionsMux[url]
	if !exists {
		mux = util.NewChanMultiplex(
			func(cm *util.ChanMultiplex[RegionUpdate], done chan struct{}) {
				fmt.Println("Starting new connection")
				err := util.FetchDataSSE(url, done, func(e sse.Event) {
					data := fmt.Sprint(e.Data)

					var cond RegionUpdate
					err := json.Unmarshal([]byte(data), &cond)
					if err != nil {
						fmt.Printf("Could not unmarshal region condition updates: %v\n", err)
						return
					}

					cm.Notify(cond)
				}, func() {
					fmt.Println("Closing connection")
					cm.Close()
				})
				if err != nil {
					fmt.Printf("Could not Fetch region condition updates: %v\n", err)
				}
			})
		regionConditionsMux[url] = mux
	}

	event_ch := mux.Subscribe()
	finished := func() {
		mux.Unsubscribe(event_ch)
	}

	return event_ch, finished
}

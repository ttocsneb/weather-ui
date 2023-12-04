package api

import (
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-contrib/sse"
	"github.com/ttocsneb/weather-ui/util"
)

func FetchLocation(conf *util.Config, latitude float64, longitude float64) (map[string]Sensor, error) {
	params := fmt.Sprintf("lat=%v&lon=%v",
		util.EncodeURIString(strconv.FormatFloat(latitude, 'f', 14, 64)),
		util.EncodeURIString(strconv.FormatFloat(longitude, 'f', 14, 64)))

	url := fmt.Sprintf("%s/location/conditions/?%v", conf.Server, params)

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

func FetchLocationUpdates(conf *util.Config, latitude float64, longitude float64) (chan RegionUpdate, func()) {
	params := fmt.Sprintf("lat=%v&lon=%v",
		util.EncodeURIString(strconv.FormatFloat(latitude, 'f', 14, 64)),
		util.EncodeURIString(strconv.FormatFloat(longitude, 'f', 14, 64)))

	url := fmt.Sprintf("%s/location/conditions/updates/?%v", conf.Server, params)

	seed := rand.Int()

	mux, exists := regionConditionsMux[url]
	fmt.Printf("%v: Finding existing location updates\n", seed)
	if !exists {
		fmt.Printf("%v: Starting location updates\n", seed)
		mux = util.NewChanMultiplex(func(cm *util.ChanMultiplex[RegionUpdate], done chan struct{}) {
			err := util.FetchDataSSE(url, done, func(e sse.Event) {
				data := fmt.Sprint(e.Data)

				fmt.Printf("%v: Received Data\n", seed)

				var cond RegionUpdate
				err := json.Unmarshal([]byte(data), &cond)
				if err != nil {
					fmt.Printf("Could not unmarshal location condition updates: %v\n", err)
					return
				}

				cm.Notify(cond)
			}, func() {
				fmt.Printf("%v: Closing location updates\n", seed)
				cm.Close()
			})
			if err != nil {
				fmt.Printf("Could not fetch location condition updates: %v\n", err)
			}
		})
		regionConditionsMux[url] = mux
	}

	event_ch := mux.Subscribe()
	finished := func() {
		fmt.Println("Trying to unsubscribe...")
		mux.Unsubscribe(event_ch)
	}

	return event_ch, finished
}

func FetchNearestStation(conf *util.Config, lat float64, lon float64) (Info, error) {
	params := fmt.Sprintf("lat=%v&lon=%v",
		util.EncodeURIString(strconv.FormatFloat(lat, 'f', 14, 64)),
		util.EncodeURIString(strconv.FormatFloat(lon, 'f', 14, 64)))

	url := fmt.Sprintf("%s/location/nearest/?%v", conf.Server, params)

	response, err := util.FetchDataToBytes(url)
	if err != nil {
		return Info{}, err
	}

	var info Info
	err = json.Unmarshal(response, &info)
	if err != nil {
		return Info{}, err
	}

	return info, nil
}

type geoResult struct {
	Status      string
	Description string
	Data        struct {
		Geo struct {
			// Host           string
			// Ip             string
			// Rdns           string
			// Asn            string
			// Isp            string
			// Country_name   string
			// Country_code   string
			// Region_name    string
			// Region_code    string
			// City           string
			// Postal_code    string
			// Continent_name string
			Latitude  float64
			Longitude float64
			// Metro_code     int
			// Timezone       string
			// Datetime       string
		}
	}
}

func EstimateLocation(conf *util.Config, req *http.Request) (float64, float64, error) {
	addr := req.Header.Get("X-Forwarded-For")
	if addr == "" {
		addr = strings.Split(req.RemoteAddr, ":")[0]
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://tools.keycdn.com/geo.json?host=%v", addr), nil)
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("keycdn-tools:%v", conf.ServerName))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, err
	}

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, 0, err
	}

	var response geoResult
	err = json.Unmarshal(content, &response)
	if err != nil {
		return 0, 0, err
	}

	fmt.Printf("Response: %v\n", response)

	return response.Data.Geo.Latitude, response.Data.Geo.Longitude, nil
}

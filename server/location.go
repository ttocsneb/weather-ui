package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/ttocsneb/weather-ui/api"
	"github.com/ttocsneb/weather-ui/util"
)

func getLocation(conf *util.Config, req *http.Request) (float64, float64, error) {
	var lat float64
	var lon float64
	var err error

	req.ParseForm()

	if req.Form.Get("estimate") == "true" {
		lat, lon, err = api.EstimateLocation(conf, req)
		if err != nil {
			return 0, 0, err
		}
	} else {
		lat, err = strconv.ParseFloat(req.Form.Get("lat"), 64)
		if err != nil {
			return 0, 0, errors.New("400 Invalid latitude")
		}
		lon, err = strconv.ParseFloat(req.Form.Get("lon"), 64)
		if err != nil {
			return 0, 0, errors.New("400 Invalid longitude")
		}
	}

	return lat, lon, nil
}

func LocationRoutes(router *mux.Router, conf *util.Config) {
	location := HandlerFuncError(func(res http.ResponseWriter, req *http.Request) error {
		vars := make(map[string]any)
		vars["Config"] = conf

		req.ParseForm()

		lat, lon, err := getLocation(conf, req)
		if err != nil {
			return err
		}

		data, err := api.FetchLocation(conf, lat, lon)
		if err != nil {
			if err.Error() == "404 Not Found" {
				res.WriteHeader(404)
				res.Write([]byte("<p>No stations in range</p>"))
				return nil
			}
			return err
		}

		vars["Conditions"] = data

		return RenderTemplate(res, "region-update.html", vars)
	})

	router.Handle("/location/conditions/", location)

	updates := HandlerFuncError(func(res http.ResponseWriter, req *http.Request) error {
		vars := make(map[string]any)
		vars["Config"] = conf

		req.ParseForm()
		lat, lon, err := getLocation(conf, req)
		if err != nil {
			return err
		}

		updates, done := api.FetchLocationUpdates(conf, lat, lon)
		defer done()

		fmt.Println("Fetching updates")

		res.Header().Set("Content-Type", "text/event-stream")
		res.Header().Set("Cache-Control", "no-cache")
		res.Header().Set("Connection", "keep-alive")
		res.Header().Set("Access-Control-Allow-Origin", "*")
		res.WriteHeader(200)
		res.(http.Flusher).Flush()

		on_done := req.Context().Done()
		for {
			select {
			case cond, success := <-updates:
				if !success {
					fmt.Println("Failure")
					return nil
				}
				fmt.Println("Received update")
				buf := util.BufPool.Get()

				vals := make(map[string]any)
				vals["Conditions"] = cond

				err := RenderTemplate(buf, "region-update.html", vals)
				if err != nil {
					util.BufPool.Put(buf)
					return err
				}

				data := string(buf.Bytes())
				message := fmt.Sprintf("data:%v\n\n", util.EscapeHtmlNewlines(data))

				res.Write([]byte(message))
				res.(http.Flusher).Flush()

				util.BufPool.Put(buf)
			case <-on_done:
				fmt.Println("Client Disconnected")
				return nil
			}
		}

	})

	router.Handle("/location/conditions/updates/", updates)

	nearest := HandlerFuncError(func(res http.ResponseWriter, req *http.Request) error {

		lat, lon, err := getLocation(conf, req)
		if err != nil {
			return err
		}

		info, err := api.FetchNearestStation(conf, lat, lon)
		if err != nil {
			return err
		}

		redirect := fmt.Sprintf("%v/station/%v/%v/", conf.Base, info.Server, info.Station)

		res.Header().Set("HX-Redirect", redirect)

		results := make(map[string]any)
		results["Url"] = redirect
		results["Name"] = fmt.Sprintf("%v %v Station", info.Station, info.District)

		vars := make(map[string]any)
		vars["Results"] = []any{results}

		err = RenderTemplate(res, "station-result.html", vars)
		if err != nil {
			return err
		}
		return nil
	})

	router.Handle("/location/nearest/", nearest)
}

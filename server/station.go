package server

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ttocsneb/weather-ui/api"
	"github.com/ttocsneb/weather-ui/util"
)

func StationRoutes(router *mux.Router, conf *util.Config) {
	station := HandlerFuncError(func(response http.ResponseWriter, request *http.Request) error {
		vars := mux.Vars(request)
		server := vars["server"]
		station := vars["station"]

		conditions, err := api.FetchStationConditions(conf, server, station)
		if err != nil {
			return err
		}
		info, err := api.FetchStationInfo(conf, server, station)
		if err != nil {
			return err
		}
		// content := string(data[:n])

		vals := make(map[string]any)
		vals["Config"] = conf
		vals["Title"] = conditions.Station
		vals["Conditions"] = conditions
		vals["Info"] = info

		err = RenderTemplate(response, "station.html", vals)

		return err
	})

	station_updates := HandlerFuncError(func(response http.ResponseWriter, request *http.Request) error {
		vars := mux.Vars(request)
		server := vars["server"]
		station := vars["station"]

		response.Header().Set("Content-Type", "text/event-stream")
		response.Header().Set("Cache-Control", "no-cache")
		response.Header().Set("Connection", "keep-alive")
		response.Header().Set("Access-Control-Allow-Origin", "*")
		response.WriteHeader(200)
		response.(http.Flusher).Flush()

		conditions, done := api.FetchStationConditionUpdates(conf, server, station)
		defer done()

		fmt.Printf("Listening for updates from %v-%v\n", server, station)

		on_done := request.Context().Done()
		for {
			select {
			case cond := <-conditions:
				buf := util.BufPool.Get()

				vals := make(map[string]any)
				vals["Conditions"] = cond

				err := RenderTemplate(buf, "station-update.html", vals)
				if err != nil {
					util.BufPool.Put(buf)
					return err
				}

				data := string(buf.Bytes())

				message := fmt.Sprintf("data:%v\n\n", data)

				response.Write([]byte(message))
				response.(http.Flusher).Flush()

				util.BufPool.Put(buf)
			case <-on_done:
				fmt.Printf("Closing Listener...\n")
				return nil
			}
		}
	})

	station_rapid := HandlerFuncError(func(response http.ResponseWriter, request *http.Request) error {
		vars := mux.Vars(request)
		server := vars["server"]
		station := vars["station"]

		response.Header().Set("Content-Type", "text/event-stream")
		response.Header().Set("Cache-Control", "no-cache")
		response.Header().Set("Connection", "keep-alive")
		response.Header().Set("Access-Control-Allow-Origin", "*")
		response.WriteHeader(200)
		response.(http.Flusher).Flush()

		conditions, done := api.FetchStationRapidConditionUpdates(conf, server, station)
		defer done()

		fmt.Printf("Listening for rapid updates from %v-%v\n", server, station)

		on_done := request.Context().Done()
		for {
			select {
			case cond := <-conditions:
				buf := util.BufPool.Get()

				vals := make(map[string]any)
				vals["Conditions"] = cond

				err := RenderTemplate(buf, "station-update.html", vals)
				if err != nil {
					util.BufPool.Put(buf)
					return err
				}

				data := string(buf.Bytes())

				message := fmt.Sprintf("data:%v\n\n", data)

				response.Write([]byte(message))
				response.(http.Flusher).Flush()
				util.BufPool.Put(buf)
			case <-on_done:
				fmt.Printf("Closing Listener...\n")
				return nil
			}
		}
	})

	router.Handle("/station/{server}/{station}/", station)
	router.Handle("/station/{server}/{station}/updates/", station_updates)
	router.Handle("/station/{server}/{station}/updates/rapid/", station_rapid)
}

package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/ttocsneb/weather-ui/api"
	"github.com/ttocsneb/weather-ui/util"
)

func RegionRoutes(router *mux.Router, conf *util.Config) {
	updates := HandlerFuncError(func(response http.ResponseWriter, request *http.Request) error {
		query := mux.Vars(request)

		country, _ := util.DecodeURIString(query["country"])
		region, _ := util.DecodeURIString(query["region"])
		city, _ := util.DecodeURIString(query["city"])
		district, _ := util.DecodeURIString(query["district"])

		conditions, done := api.FetchRegionUpdates(conf, country, region, city, district)
		defer done()

		response.Header().Set("Content-Type", "text/event-stream")
		response.Header().Set("Cache-Control", "no-cache")
		response.Header().Set("Connection", "keep-alive")
		response.Header().Set("Access-Control-Allow-Origin", "*")
		response.WriteHeader(200)
		response.(http.Flusher).Flush()

		on_done := request.Context().Done()
		for {
			select {
			case cond := <-conditions:
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

				response.Write([]byte(message))
				response.(http.Flusher).Flush()

				util.BufPool.Put(buf)
			case <-on_done:
				fmt.Printf("Closing Listener...\n")
				return nil
			}
		}
	})

	router.Handle("/region/{country}/{region}/{city}/updates/", updates)
	router.Handle("/region/{country}/{region}/{city}/{district}/updates/", updates)

	region := HandlerFuncError(func(response http.ResponseWriter, request *http.Request) error {
		query := mux.Vars(request)

		country, _ := util.DecodeURIString(query["country"])
		region, _ := util.DecodeURIString(query["region"])
		city, _ := util.DecodeURIString(query["city"])
		district, _ := util.DecodeURIString(query["district"])

		values, err := api.FetchRegion(conf, country, region, city, district)
		if err != nil {
			return err
		}

		vars := make(map[string]any)

		vars["Config"] = conf
		vars["Conditions"] = values
		vars["Country"] = country
		vars["Region"] = region
		vars["City"] = city
		vars["District"] = district

		return RenderTemplate(response, "region.html", vars)
	})

	router.Handle("/region/{country}/{region}/{city}/", region)
	router.Handle("/region/{country}/{region}/{city}/{district}/", region)

	search := HandlerFuncError(func(response http.ResponseWriter, request *http.Request) error {

		if request.Method != "POST" {
			response.WriteHeader(403)
			response.Write([]byte("403 Not Authorized"))
			return nil
		}

		vars := make(map[string]any)
		vars["Config"] = conf

		request.ParseForm()

		if request.Form.Has("query") {
			query := request.Form.Get("query")
			vars["Query"] = query

			segments := strings.Split(query, ",")
			for i, v := range segments {
				segments[i] = strings.TrimSpace(v)
			}

			fmt.Printf("Searching for %v\n", segments)

			results, err := api.SearchRegion(conf, segments...)
			if err != nil {
				return err
			}

			if len(results) == 0 {
				vars["Bad"] = true
			}

			new_results := []api.Region{}
			for _, region := range results {
				city := api.Region{
					Country:  region.Country,
					Region:   region.Region,
					City:     region.City,
					District: "",
				}
				if !util.Contains(new_results, &city) {
					new_results = append(new_results, city)
				}
			}

			fmt.Println(results)
			fmt.Println(new_results)

			results = append(new_results, results...)
			fmt.Println(results)

			// if len(results) == 1 {
			// 	http.Redirect(response, request,
			// 		fmt.Sprintf("%v/region/%v/%v/%v/%v/", conf.Base,
			// 			util.EncodeURIString(results[0].Country),
			// 			util.EncodeURIString(results[0].Region),
			// 			util.EncodeURIString(results[0].City),
			// 			util.EncodeURIString(results[0].District)), 302)
			// 	return nil
			// }

			// if len(results) > 1 {
			// 	city := [3]string{results[0].Country, results[0].Region, results[0].City}
			// 	valid := true
			//
			// 	for _, region := range results {
			// 		foo := [3]string{region.Country, region.Region, region.City}
			// 		if city != foo {
			// 			valid = false
			// 			break
			// 		}
			// 	}
			// 	if valid {
			// 		http.Redirect(response, request,
			// 			fmt.Sprintf("%v/region/%v/%v/%v/", conf.Base,
			// 				util.EncodeURIString(city[0]),
			// 				util.EncodeURIString(city[1]),
			// 				util.EncodeURIString(city[2])), 302)
			// 	}
			// }

			vars["Results"] = results
		}

		return RenderTemplate(response, "region-search.html", vars)
	})

	router.Handle("/region/search/", search)

}

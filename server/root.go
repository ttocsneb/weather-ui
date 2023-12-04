package server

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gorilla/mux"
	"github.com/ttocsneb/weather-ui/api"
	"github.com/ttocsneb/weather-ui/util"
)

func RootRoutes(router *mux.Router, conf *util.Config) {
	root := HandlerFuncError(func(res http.ResponseWriter, req *http.Request) error {
		vars := make(map[string]any)
		vars["Config"] = conf

		vars["Method"] = req.Method
		if req.Method == "POST" {
			req.ParseForm()

			if req.Form.Has("query") {
				query := req.Form.Get("query")
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

				if len(results) == 1 {
					http.Redirect(res, req,
						fmt.Sprintf("%v/region/%v/%v/%v/%v/", conf.Base,
							util.EncodeURIString(results[0].Country),
							util.EncodeURIString(results[0].Region),
							util.EncodeURIString(results[0].City),
							util.EncodeURIString(results[0].District)), 302)
					return nil
				}

				if len(results) > 1 {
					city := [3]string{results[0].Country, results[0].Region, results[0].City}
					valid := true

					for _, region := range results {
						foo := [3]string{region.Country, region.Region, region.City}
						if city != foo {
							valid = false
							break
						}
					}
					if valid {
						http.Redirect(res, req,
							fmt.Sprintf("%v/region/%v/%v/%v/", conf.Base,
								util.EncodeURIString(city[0]),
								util.EncodeURIString(city[1]),
								util.EncodeURIString(city[2])), 302)
					}
				}

				vars["Results"] = results
			}
		}

		return RenderTemplate(res, "root.html", vars)
	})

	router.Handle("/", root)
}

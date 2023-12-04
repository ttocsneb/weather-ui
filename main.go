package main

import (
	"os"

	"github.com/ttocsneb/weather-ui/api"
	"github.com/ttocsneb/weather-ui/server"
	"github.com/ttocsneb/weather-ui/util"
)

func main() {
	util.Setup()
	api.Setup()

	conf_path := "config.toml"
	if len(os.Args) > 1 {
		conf_path = os.Args[1]
	}

	conf, err := util.ParseConfig(conf_path)
	if err != nil {
		panic(err)
	}

	panic(server.Serve(conf))
}

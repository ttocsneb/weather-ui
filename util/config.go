package util

import (
	"os"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Server     string
	Base       string
	Port       uint16
	ServerName string
}

func ParseConfig(path string) (Config, error) {
	var conf Config
	conf.Port = 8080
	f, err := os.ReadFile(path)
	if err != nil {
		return conf, err
	}
	_, err = toml.Decode(string(f), &conf)
	return conf, err
}

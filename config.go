package main

import (
	"errors"
	"github.com/spf13/viper"
	"log"
	"strings"
)

type config struct {
	RPC_ENDPOINT             string
	RPC_RATE_LIMITER_NUMBER  float64
	GIN_SERVER_ADDRESS       string
	GIN_TRUSTED_PROXIES_LIST []string
}

func loadConfig(envPath string) (*config, error) {
	viper.SetConfigFile(envPath)
	viper.AutomaticEnv()
	err := viper.ReadInConfig()
	if err != nil {
		log.Println("Can't read config file: ", envPath)
		log.Println("Loading config from environment variables...")
	}
	//todo check if the config is not nil and valid
	cfg := &config{}
	cfg.RPC_ENDPOINT = viper.GetString("RPC_ENDPOINT")
	cfg.RPC_RATE_LIMITER_NUMBER = viper.GetFloat64("RPC_RATE_LIMITER_NUMBER")
	cfg.GIN_SERVER_ADDRESS = viper.GetString("GIN_SERVER_ADDRESS")
	cfg.GIN_TRUSTED_PROXIES_LIST = strings.Split(viper.GetString("GIN_TRUSTED_PROXIES_LIST"), ",")

	if cfg.RPC_ENDPOINT == "" || cfg.RPC_RATE_LIMITER_NUMBER == 0 || cfg.GIN_SERVER_ADDRESS == "" || len(cfg.GIN_TRUSTED_PROXIES_LIST) == 0 {
		return nil, errors.New("config is not valid")
	}

	return cfg, nil
}

package clyper

import "github.com/spf13/viper"

type ServerConfig struct {
	Listen string `mapstructure:"listen"`
}

type Config struct {
	Server ServerConfig `mapstructure:"server"`
}

func LoadConfig() (*Config, error) {
	var config Config
	err := viper.Unmarshal(&config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}

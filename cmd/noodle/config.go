package main

type Endpoint struct {
	Address string `mapstructure:"address"`
	Port    int    `mapstructure:"port"`
}

func NewConfig() *Config {
	return &Config{}
}

type Config struct {
	Endpoints []Endpoint
}

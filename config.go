package main

import (
	"gopkg.in/yaml.v3"
	"os"
	"path/filepath"
)

var (
	config Config
)

type MysqlConfig struct {
	Addr     string `yaml:"Addr"`
	Username string `yaml:"Username"`
	Password string `yaml:"Password"`
	Database string `yaml:"Database"`
}

type RedisConfig struct {
	Addr string `yaml:"Addr"`
}

type ServerConfig struct {
	Addr string `yaml:"Addr"`
}

type TranscodeConfig struct {
	FFmpegPath     string `yaml:"FFmpegPath"`
	FFprobePath    string `yaml:"FFprobePath"`
	TranscodeCount int64  `yaml:"TranscodeCount"`
}

type Config struct {
	Mysql     MysqlConfig     `yaml:"Mysql"`
	Redis     RedisConfig     `yaml:"Redis"`
	Transcode TranscodeConfig `yaml:"Transcode"`
	Server    ServerConfig    `yaml:"Server"`
}

func configInit() {
	filename, _ := filepath.Abs("config.yaml")
	yamlFile, err := os.ReadFile(filename)

	if err != nil {
		panic(err)
	}

	err = yaml.Unmarshal(yamlFile, &config)

	if err != nil {
		panic(err)
	}
}

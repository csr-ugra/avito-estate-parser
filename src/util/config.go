package util

import (
	"errors"
	"fmt"
	_ "github.com/joho/godotenv/autoload"
	"log"
	"os"
)

type configValue struct {
	envVarName   string
	required     bool
	errorMessage string
	Value        string
}

type Config struct {
	DevtoolsWebsocketUrl configValue
	DbConnectionString   configValue
}

func NewConfig() *Config {
	const devtoolsWebsocketUrlName = "DEVTOOLS_WEBSOCKET_URL"
	const dbConnectionString = "DEVTOOLS_WEBSOCKET_URL"
	return &Config{
		DevtoolsWebsocketUrl: configValue{
			envVarName:   devtoolsWebsocketUrlName,
			required:     true,
			errorMessage: fmt.Sprintf("environment variable %s is not set", devtoolsWebsocketUrlName),
		},
		DbConnectionString: configValue{
			envVarName:   dbConnectionString,
			required:     true,
			errorMessage: fmt.Sprintf("make sure that env variable %s is set and in DSN format", dbConnectionString),
		}}
}

var config *Config

func GetConfig() *Config {
	if config == nil {
		return load()
	}

	return config
}

func load() *Config {
	config := NewConfig()

	if err := populateEnv(config.DevtoolsWebsocketUrl); err != nil {
		log.Fatal(err)
	}
	if err := populateEnv(config.DbConnectionString); err != nil {
		log.Fatal(err)
	}

	return config
}

func populateEnv(m configValue) (err error) {
	v := os.Getenv(m.envVarName)

	if v == "" && m.required {
		return errors.New(m.errorMessage)
	}

	m.Value = v
	return nil
}

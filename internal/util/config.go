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
	defaultValue string
	Value        string
}

type Config struct {
	DevtoolsWebsocketUrl configValue
	DbConnectionString   configValue
	SeqUrl               configValue
	SeqToken             configValue
	Environment          configValue
}

func NewConfig() *Config {
	const devtoolsWebsocketUrlName = "DEVTOOLS_WEBSOCKET_URL"
	const dbConnectionStringName = "DB_CONNECTION_STRING"
	const seqUrlName = "SEQ_URL"
	const seqTokenName = "SEQ_TOKEN"
	const environmentName = "ENVIRONMENT"

	return &Config{
		DevtoolsWebsocketUrl: configValue{
			envVarName:   devtoolsWebsocketUrlName,
			required:     false,
			defaultValue: "ws://127.0.0.1:7317",
		},
		DbConnectionString: configValue{
			envVarName:   dbConnectionStringName,
			required:     true,
			errorMessage: fmt.Sprintf("make sure that environment variable %s is set and in DSN format", dbConnectionStringName),
		},
		SeqUrl: configValue{
			envVarName: seqUrlName,
			required:   false,
		},
		SeqToken: configValue{
			envVarName: seqTokenName,
			required:   false,
		},
		Environment: configValue{
			envVarName:   environmentName,
			required:     false,
			defaultValue: "development",
		},
	}
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

	if err := populateEnv(&config.DevtoolsWebsocketUrl); err != nil {
		log.Fatal(err)
	}
	if err := populateEnv(&config.DbConnectionString); err != nil {
		log.Fatal(err)
	}
	if err := populateEnv(&config.SeqUrl); err != nil {
		log.Fatal(err)
	}
	if err := populateEnv(&config.SeqToken); err != nil {
		log.Fatal(err)
	}
	if err := populateEnv(&config.Environment); err != nil {
		log.Fatal(err)
	}

	return config
}

func populateEnv(m *configValue) (err error) {
	v := os.Getenv(m.envVarName)

	if v == "" && m.required {
		if m.errorMessage != "" {
			return errors.New(m.errorMessage)
		}

		return fmt.Errorf("environment variable %s is not set", m.envVarName)
	}

	if v == "" && m.defaultValue != "" {
		m.Value = m.defaultValue
	}

	m.Value = v
	return nil
}

package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"grail/sysinfra/cfg/log"
	"io/ioutil"
	"os"
)

const (
	LOG_LEVEL    = "LOG_LEVEL"
	BRANCH       = "BRANCH"
	BUILD_NUMBER = "BUILD_NUMBER"
	COMMIT       = "COMMIT"
	VERSION      = "VERSION"
)

type buildData struct {
	Version     string `json:"version,omitempty" default:"1.3" env:"VERSION"`
	Commit      string `json:"commit,omitempty" env:"COMMIT"`
	Branch      string `json:"branch,omitempty" env:"BRANCH"`
	BuildNumber string `json:"build_number,omitempty" env:"BUILD_NUMBER"`
}

// Build contains configuration data about the application version
type Build struct {
	buildData
}

type Configuration struct {
	LogLevel string `json:"log_level" env:"LOG_LEVEL"`
	Build    Build  `json:"build"`
}

var defaultConfiguration = Configuration{
	LogLevel: "INFO",
	Build: Build{
		buildData: buildData{
			Version: "1.0.0",
		},
	},
}

var configurationData Configuration

// Config returns the configuration data
func Config() *Configuration {
	return &configurationData
}

type initOptions struct {
	DefaultValues map[string]string
}

// Set is a functional argument that you can pass to Defaults to set a default configuration value.
// Key should be in environment variable format (e.g. DATASOURCE_HOST)
func Set(key string, value string) func(*initOptions) {
	return func(o *initOptions) {
		if o.DefaultValues == nil {
			o.DefaultValues = make(map[string]string)
		}
		o.DefaultValues[key] = value
	}
}

// Defaults is a functional argument you can pass to Init(). It's arguments would be one or more
// calls to Set()
func Defaults(setters ...func(*initOptions)) func(*initOptions) {
	return func(o *initOptions) {
		for _, setter := range setters {
			setter(o)
		}
	}
}

// Init initializes the configuration module. It accepts zero or more functional arguments. Use
// Defaults to specify a list of application defaults and EnableREST to register REST endpoints
// for the configuration. For example:
//     Init(Defaults(Set("DATASOURCE_HOST", "localhost")))
//     Init(EnableREST)
func Init(options ...func(*initOptions)) (*Configuration, error) {
	InitFromConfigFiles()

	ops := initOptions{}
	for _, option := range options {
		option(&ops)
	}
	configurationData = defaultConfiguration

	// set default values
	for key, value := range ops.DefaultValues {
		DefaultMapProvider.Set(key, value)
	}

	// apply data from external data providers
	err := ApplyExternalConfig(&configurationData, 4)
	if err != nil {
		return nil, fmt.Errorf("error resolving config values: %v", err)
	}
	var level log.Level
	level.UnmarshalText([]byte(configurationData.LogLevel))
	log.SetLevel(level)
	b, err := json.Marshal(configurationData)
	if err != nil {
		log.Infof("Configuration: %s", string(b))
	}

	return &configurationData, nil
}

// UpdateFromJSON merges any data from the specified json structure into the current configuration.
// Fields that are missing in the JSON data will retain their previous value.
func UpdateFromJSON(jsonData string, obj interface{}) error {
	err := json.Unmarshal([]byte(jsonData), obj)
	return err
}

func InitFromConfigFiles() {
	initFromConfigFile("./config.json")
}

//Read configuration file
func initFromConfigFile(filePath string) {
	if stat, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		log.Infof("Config file does not exist")
		return
	} else if stat.IsDir() {
		log.Infof("Config file path is a directory")
		return
	}

	// Open our jsonFile
	jsonFile, err := os.Open(filePath)
	// if we os.Open returns an error then handle it
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("Successfully Opened %s", filePath)
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	// read our opened jsonFile as a byte array.
	byteValue, err := ioutil.ReadAll(jsonFile)
	if err != nil {
		log.Println(err)
		return
	}

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'users' which we defined above
	err = json.Unmarshal(byteValue, &defaultConfiguration)
	if err != nil {
		log.Println(err)
		return
	}
	log.Printf("Default Config is %v", defaultConfiguration)
}

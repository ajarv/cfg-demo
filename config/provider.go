package config

import (
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"grail/sysinfra/cfg/log"
)

// KeyValueProvider defines the interface that must be satisfied by all Providers
type KeyValueProvider interface {
	Get(key string) (string, error)
}

// EnvironmentProvider is used to update the configuration from environment variables
type EnvironmentProvider struct{}

// Get fetches the environment variable value for the specified key. An empty string
// is returned if not found.
func (e EnvironmentProvider) Get(key string) (string, error) {
	return os.Getenv(key), nil
}

// MapProvider is used to update the configuration from a map that has been initialized
// by the application
type MapProvider struct {
	store map[string]string
}

// Get fetches the mapped value for the specified key. An empty string
// is returned if not found.
func (m *MapProvider) Get(key string) (string, error) {
	if m.store == nil {
		m.store = make(map[string]string)
	}
	return m.store[key], nil
}

// Set sets a value in the map. The key should match the field's environment variable name.
func (m *MapProvider) Set(key string, value string) {
	if m.store == nil {
		m.store = make(map[string]string)
	}
	m.store[key] = value
}

// DefaultMapProvider is the built in map provider
var DefaultMapProvider = &MapProvider{}

// default data providers in decreasing order of priority
var dataProviders = []KeyValueProvider{&EnvironmentProvider{}, DefaultMapProvider}

// ClearDataProviders removes all configuration data providers
func ClearDataProviders() {
	dataProviders = nil
}

// AddDataProvider adds a new KeyValueProvider to the list of providers
// that are checked for configuration values. The list is in descending order
// of priority.
func AddDataProvider(p KeyValueProvider) {
	dataProviders = append(dataProviders, p)
}

// ApplyExternalConfig walks through the specified configuration data structure and
// updates the configuration fields from the configured data providers
func ApplyExternalConfig(s interface{}, maxDepth int) error {
	walkStruct(reflect.ValueOf(s).Elem(), maxDepth)
	return nil
}

// getValue searches the configured providers for the highest priority provider
// that has a value available for the specified key.
// nolint:unused,deadcode
func getValue(key string) string {
	for _, prov := range dataProviders {
		v, err := prov.Get(key)
		if err == nil && v != "" {
			return v
		}
	}
	return ""
}

// getValueDefault searches the configured providers for the highest priority provider
// that has a value available for the specified key. Else return envDefault if it is non-empty on
// failure of all searhces
func getValueDefault(key string, envDefault string) string {
	for _, prov := range dataProviders {
		v, err := prov.Get(key)
		if err == nil && v != "" {
			return v
		}
	}
	return envDefault
}

var setters map[reflect.Kind]func(field reflect.Value, value string)

// initSetters creates setters for many built in data types
func initSetters() {
	// adapted from https://github.com/mcuadros/go-defaults (MIT)
	setters = make(map[reflect.Kind]func(field reflect.Value, value string))
	setters[reflect.Bool] = func(field reflect.Value, value string) {
		val, err := strconv.ParseBool(value)
		if err == nil {
			field.SetBool(val)
		}
	}

	setters[reflect.Int] = func(field reflect.Value, value string) {
		val, err := strconv.ParseInt(value, 10, 64)
		if err == nil {
			field.SetInt(val)
		}
	}

	setters[reflect.Int8] = setters[reflect.Int]
	setters[reflect.Int16] = setters[reflect.Int]
	setters[reflect.Int32] = setters[reflect.Int]
	setters[reflect.Int64] = setters[reflect.Int]

	setters[reflect.Float32] = func(field reflect.Value, value string) {
		val, err := strconv.ParseFloat(value, 64)
		if err == nil {
			field.SetFloat(val)
		}
	}

	setters[reflect.Float64] = setters[reflect.Float32]

	setters[reflect.Uint] = func(field reflect.Value, value string) {
		val, err := strconv.ParseUint(value, 10, 64)
		if err == nil {
			field.SetUint(val)
		}
	}

	setters[reflect.Uint8] = setters[reflect.Uint]
	setters[reflect.Uint16] = setters[reflect.Uint]
	setters[reflect.Uint32] = setters[reflect.Uint]
	setters[reflect.Uint64] = setters[reflect.Uint]

	setters[reflect.String] = func(field reflect.Value, value string) {
		val := parseDateTimeString(value)
		field.SetString(val)
	}
}
func parseDateTimeString(data string) string {

	pattern := regexp.MustCompile(`\{\{(\w+\:(?:-|)\d*,(?:-|)\d*,(?:-|)\d*)\}\}`)
	matches := pattern.FindAllStringSubmatch(data, -1) // matches is [][]string
	for _, match := range matches {

		tags := strings.Split(match[1], ":")
		if len(tags) == 2 {

			valueStrings := strings.Split(tags[1], ",")
			if len(valueStrings) == 3 {
				var values [3]int
				for key, valueString := range valueStrings {
					num, _ := strconv.ParseInt(valueString, 10, 64)
					values[key] = int(num)
				}

				switch tags[0] {

				case "date":
					str := time.Now().AddDate(values[0], values[1], values[2]).Format("2006-01-02")
					data = strings.Replace(data, match[0], str, -1)
					break
				case "time":
					str := time.Now().Add((time.Duration(values[0]) * time.Hour) +
						(time.Duration(values[1]) * time.Minute) +
						(time.Duration(values[2]) * time.Second)).Format("15:04:05")
					data = strings.Replace(data, match[0], str, -1)
					break
				}
			}
		}

	}
	return data
}
func isStruct(fv reflect.Value, ft reflect.StructField) bool {
	return ft.Type.PkgPath() != "" && fv.Kind() == reflect.Struct
}

// setValue sets the value of the specified field to the specified value
func setValue(field reflect.Value, fieldType reflect.StructField, value string) {
	if !(field.IsValid() && field.CanSet()) {
		return
	}
	if setters == nil {
		initSetters()
	}
	kind := fieldType.Type.Kind()
	setter := setters[kind]
	if setter != nil {
		setter(field, value)
	} else {
		log.Warnf("did not find setter for field %s kind = %d", fieldType.Name, int(kind))
	}
}

func walkStruct(v reflect.Value, maxDepth int) {
	t := v.Type()
	log.Debugf("walk: %s %d", t.Name(), maxDepth)

	for i := 0; i < v.NumField(); i++ {
		fv := v.Field(i)
		ft := t.Field(i)
		log.Debugf("walk[%d]: %s %s", i, t.Field(i).Name, t.Field(i).Type.Name())
		// Get the field's tag value
		tag := ft.Tag.Get("env")

		if tag == "" {
			if maxDepth > 0 && isStruct(fv, ft) {
				walkStruct(fv, maxDepth-1)
			}
			continue
		}

		//log.Printf("found tag %s for field %s\n", tag, ft.Name)
		defaultTag := ft.Tag.Get("default")
		if envValue := getValueDefault(tag, defaultTag); envValue != "" {
			log.Debugf("setting %s to %s", ft.Name, envValue)
			setValue(fv, ft, envValue)
		}
	}
}

package server

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"reflect"

	"github.com/tailscale/hujson"
)

// Config holds the current configuration
type Config struct {
	Address     string `comment:"Server address and port"`
	DatabaseURL string `comment:"Database URL (default: postgres://localhost:5432)"`
	AllowAnon   bool   `comment:"Allow unauthenticated connections"`
	JWTSecret   string `comment:"Secret for JWT tokens"`
	EnableAdmin bool   `comment:"Enable administration of databases and tables"`
}

func WriteConfig(config any) ([]byte, error) {
	json := WriteObject(config)
	b := hujson.Value{Value: json}.Pack()
	return hujson.Format(b)
}

func WriteObject(o any) *hujson.Object {
	var obj hujson.Object
	v := reflect.Indirect(reflect.ValueOf(o))
	t := v.Type()
	fields := reflect.VisibleFields(t)
	for i, field := range fields {
		comment := hujson.Extra("\n" + `// ` + field.Tag.Get("comment") + "\n")
		name := hujson.String(field.Name)
		var value hujson.ValueTrimmed
		switch field.Type.Kind() {
		case reflect.Struct:
			value = WriteObject(v.Field(i).Interface())
		case reflect.String:
			value = hujson.String(v.Field(i).String())
		case reflect.Int:
			value = hujson.Int(v.Field(i).Int())
		case reflect.Float32, reflect.Float64:
			value = hujson.Float(v.Field(i).Float())
		case reflect.Bool:
			value = hujson.Bool(v.Field(i).Bool())
		}
		obj.Members = append(obj.Members, hujson.ObjectMember{
			Name: hujson.Value{
				BeforeExtra: comment,
				Value:       name,
			},
			Value: hujson.Value{
				Value: value,
			},
		})
	}
	return &obj
}

// GetConfig loads the configuration
func GetConfig(configFile string) *Config {

	// Defaults
	config := &Config{
		Address:     ":8081",
		DatabaseURL: "postgres://localhost:5432/",
		AllowAnon:   false,
	}

	// Command line flags
	flag.StringVar(&config.Address, "addr", config.Address, "Address")
	flag.StringVar(&config.DatabaseURL, "dburl", config.DatabaseURL, "DatabaseURL")

	// Read config file
	b, err := ioutil.ReadFile(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error reading the configuration file (%s)", err)
			return nil
		}
	} else {
		b, err = hujson.Standardize(b)
		if err != nil {
			log.Printf("Invalid configuration file (%s)", err)
		} else {
			err = json.Unmarshal(b, config)
			if err != nil {
				log.Printf("Invalid configuration file (%s)", err)
			}
		}
	}
	// Update config file
	b, _ = WriteConfig(config)
	err = ioutil.WriteFile(configFile, b, 0777)
	if err != nil {
		log.Printf("Error writing the configuration file (%s)", err)
	}
	flag.Parse()
	return config
}

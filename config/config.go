package config

import (
	"encoding/json"
	"flag"
	"io/ioutil"
	"log"
	"os"
	"reflect"

	"github.com/tailscale/hujson"
)

type Database struct {
	URL                string `comment:"Database URL (default: postgres://localhost:5432)"`
	MinPoolConnections uint   `comment:"Miminum connections per pool (default: 10)"`
	MaxPoolConnections uint   `comment:"Maximum connections per pool (default: 100)"`
	AuthRole           string `comment:"Authorization role (default: auth)"`
	AnonRole           string `comment:"Anonymous role (default: anon)"`
	AllowedDatabases   []string
	SchemaSearchPath   []string
}

// Config holds the current configuration
type Config struct {
	Address     string   `comment:"Server address and port (default localhost:8081)"`
	AllowAnon   bool     `comment:"Allow unauthenticated connections"`
	JWTSecret   string   `comment:"Secret for JWT tokens"`
	EnableAdmin bool     `comment:"Enable administration of databases and tables"`
	Database    Database `comment:"Database configuration"`
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
		case reflect.Uint:
			value = hujson.Uint(v.Field(i).Uint())
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

func DefaultDatabaseConfig() *Database {
	return &Database{
		URL:                "postgres://localhost:5432/",
		MinPoolConnections: 10,
		MaxPoolConnections: 100,
		AuthRole:           "auth",
		AnonRole:           "anon",
	}
}

func DefaultConfig() *Config {
	return &Config{
		Address:     ":8081",
		AllowAnon:   false,
		JWTSecret:   "",
		EnableAdmin: false,
		Database:    *DefaultDatabaseConfig(),
	}
}

// GetConfig loads the configuration
func GetConfig(configFile string) *Config {

	// Defaults
	config := DefaultConfig()

	// Command line flags
	flag.StringVar(&config.Address, "addr", config.Address, "Address")
	flag.StringVar(&config.Database.URL, "dburl", config.Database.URL, "DatabaseURL")

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

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
	URL                string   `comment:"Database URL (default: postgres://localhost:5432)"`
	MinPoolConnections int32    `comment:"Miminum connections per pool (default: 10)"`
	MaxPoolConnections int32    `comment:"Maximum connections per pool (default: 100)"`
	AuthRole           string   `comment:"Authorization role (default: auth)"`
	AnonRole           string   `comment:"Anonymous role (default: anon)"`
	AllowedDatabases   []string `comment:"Allowed databases (default: [] for all)"`
	SchemaSearchPath   []string `comment:"Schema search path (default: [] for default Postgres search path)"`
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
	fields := reflect.VisibleFields(v.Type())
	var value hujson.ValueTrimmed
	for i, structfield := range fields {
		comment := hujson.Extra("\n" + `// ` + structfield.Tag.Get("comment") + "\n")
		name := hujson.String(structfield.Name)
		field := v.Field(i)
		switch structfield.Type.Kind() {
		case reflect.Struct:
			value = WriteObject(field.Interface())
		case reflect.Slice:
			value = WriteArray(field.Interface())
		case reflect.String:
			value = hujson.String(field.String())
		case reflect.Int, reflect.Int32, reflect.Int64:
			value = hujson.Int(field.Int())
		case reflect.Uint, reflect.Uint32, reflect.Uint64:
			value = hujson.Uint(field.Uint())
		case reflect.Float32, reflect.Float64:
			value = hujson.Float(field.Float())
		case reflect.Bool:
			value = hujson.Bool(field.Bool())
		}
		obj.Members = append(obj.Members, hujson.ObjectMember{
			Name: hujson.Value{
				BeforeExtra: comment,
				Value:       name,
			},
			Value: hujson.Value{Value: value},
		})
	}
	return &obj
}

func WriteArray(a any) *hujson.Array {
	var array hujson.Array
	v := reflect.ValueOf(a)
	var value hujson.ValueTrimmed
	for i := 0; i < v.Len(); i++ {
		item := v.Index(i)
		switch item.Type().Kind() {
		case reflect.String:
			value = hujson.String(item.String())
		}
		array.Elements = append(array.Elements, hujson.Value{Value: value})
	}
	return &array
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

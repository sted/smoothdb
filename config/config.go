package config

import (
	"encoding/json"
	"log"
	"os"
	"reflect"

	"github.com/tailscale/hujson"
)

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

// GetConfig loads the configuration
func GetConfig[C any](config *C, configFile string) *C {

	// Read config file
	b, err := os.ReadFile(configFile)
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
	err = os.WriteFile(configFile, b, 0777)
	if err != nil {
		log.Printf("Error writing the configuration file (%s)", err)
	}
	return config
}

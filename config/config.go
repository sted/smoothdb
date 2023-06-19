package config

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"

	"github.com/tailscale/hujson"
)

func writeConfig(config any) ([]byte, error) {
	json := writeObject(config)
	b := hujson.Value{Value: json}.Pack()
	return hujson.Format(b)
}

func writeObject(o any) *hujson.Object {
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
			value = writeObject(field.Interface())
		case reflect.Slice:
			value = writeArray(field.Interface())
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

func writeArray(a any) *hujson.Array {
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

// GetConfig reads the configuration file
func GetConfig(config any, configFile string) error {
	b, err := os.ReadFile(configFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("error reading the configuration file (%w)", err)
		}
	} else {
		b, err = hujson.Standardize(b)
		if err != nil {
			return fmt.Errorf("invalid configuration file (%w)", err)
		} else {
			err = json.Unmarshal(b, config)
			if err != nil {
				return fmt.Errorf("invalid configuration file (%w)", err)
			}
		}
	}
	return nil
}

// SaveConfig saves the configuration file
func SaveConfig(config any, configFile string) error {
	b, err := writeConfig(config)
	if err != nil {
		return fmt.Errorf("error writing the configuration file (%w)", err)
	}
	err = os.WriteFile(configFile, b, 0777)
	if err != nil {
		return fmt.Errorf("error writing the configuration file (%w)", err)
	}
	return nil
}

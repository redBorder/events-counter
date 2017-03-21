// Simple utility for counting messages on a Kafka topic.
//
// Copyright (C) 2017 ENEO Tecnologia SL
// Author: Diego Fernández Barrear <bigomby@gmail.com>
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"errors"
	"reflect"
	"strconv"

	"github.com/Sirupsen/logrus"

	yaml "gopkg.in/yaml.v2"
)

// AppConfig contains the main application configuration.
type AppConfig struct {
	Counters struct {
		BatchTimeoutSeconds uint   `yaml:"batch_timeout_s" default:"5"`
		BatchMaxMessages    uint   `yaml:"batch_max_messages" default:"1000"`
		UUIDKey             string `yaml:"uuid_key" mandatory:"true"`
		Kafka               struct {
			ReadTopics      []string          `yaml:"read_topics" mandatory:"true"`
			WriteTopic      string            `yaml:"write_topic" mandatory:"true"`
			Attributes      map[string]string `yaml:"attributes"`
			TopicAttributes map[string]string `yaml:"topic_attributes"`
		}
	}

	// Monitor struct {
	// 	Timer struct {
	// 		Period uint `yaml:"period" default:"86400"`
	// 		Offset uint `yaml:"offset" default:"0"`
	// 	}
	// 	Kafka struct {
	// 		ReadTopics      []string          `yaml:"read_topics"`
	// 		WriteTopic      string            `yaml:"write_topic"`
	// 		Attributes      map[string]string `yaml:"attributes"`
	// 		TopicAttributes map[string]string `yaml:"topic_attributes"`
	// 	}
	// }

	// Limits struct {
	// 	UUIDS []struct {
	// 		UUID      string `yaml:"uuid"`
	// 		LimitType string `yaml:"type"`
	// 		Limit     int    `yaml:"limit"`
	// 	}
	// }
}

// verify checks for fields with "mandatory" struc tag set to "true" and if
// the field does not have a value set will fail.
func (config *AppConfig) verify() error {
	values, fields := deepFields(config)

	for i := range fields {
		if !values[i].IsValid() {
			return errors.New("Invalid field")
		}

		if !values[i].CanSet() {
			return errors.New("Can't set field for " + fields[i].Name)
		}

		if tag := fields[i].Tag.Get("mandatory"); tag != "" {
			mandatory, err := strconv.ParseBool(tag)
			if err != nil {
				return errors.New("Invalid mandatory value " + tag)
			}

			if mandatory {
				switch values[i].Kind() {
				case reflect.String:
					if values[i].String() == "" {
						return errors.New("Field \"" + fields[i].Name + "\" must be provided")
					}

				case reflect.Int:
					if values[i].Int() == 0 {
						return errors.New("Field \"" + fields[i].Name + "\" must be provided")
					}

				case reflect.Uint:
					if values[i].Uint() == 0 {
						return errors.New("Field \"" + fields[i].Name + "\" must be provided")
					}
				}
			}
		}
	}

	return nil
}

// setDefaults looks for struct tags "default="<val>"" and set value of the fields
// to <val> if does not have a previous value.
func (config *AppConfig) setDefaults() error {
	values, fields := deepFields(config)

	for i := range fields {
		if !values[i].IsValid() {
			return errors.New("Invalid field")
		}

		if !values[i].CanSet() {
			return errors.New("Can't set field for " + fields[i].Name)
		}

		if tag := fields[i].Tag.Get("default"); tag != "" {
			setDefaultField(values[i], fields[i], tag)
		}
	}

	return nil
}

// ParseConfig parse a YAML formatted string and returns a AppConfig struct
// containing the parsed configuration.
func ParseConfig(raw []byte) (*AppConfig, error) {
	config := &AppConfig{}
	err := yaml.Unmarshal(raw, &config)
	if err != nil {
		return config, errors.New("Error: " + err.Error())
	}

	if err := config.setDefaults(); err != nil {
		logrus.Fatal("Error reading configuration: " + err.Error())
	}
	if err := config.verify(); err != nil {
		logrus.Fatal("Error reading configuration: " + err.Error())
	}

	return config, nil
}

func setDefaultField(v reflect.Value, t reflect.StructField, value string) {
	switch v.Kind().String() {
	case "int":
		if defaultValue, err := strconv.ParseInt(value, 10, 64); v.Int() == 0 && err == nil {
			logrus.Warnf("Defaulting \"%s\" to %d", t.Name, defaultValue)
			v.SetInt(defaultValue)
		}

	case "uint":
		if defaultValue, err := strconv.ParseUint(value, 10, 64); v.Uint() == 0 && err == nil {
			logrus.Warnf("Defaulting \"%s\" to %d", t.Name, defaultValue)
			v.SetUint(defaultValue)
		}

	case "string":
		if len(value) > 0 {
			logrus.Warnf("Defaulting \"%s\" to %s", t.Name, value)
			v.SetString(value)
		}

	default:
		logrus.Warnln(v.Kind().String())
	}
}

func deepFields(iface interface{}) ([]reflect.Value, []reflect.StructField) {
	values := make([]reflect.Value, 0)
	fields := make([]reflect.StructField, 0)

	elem := reflect.ValueOf(iface)
	for elem.Kind() == reflect.Ptr {
		elem = elem.Elem()
	}

	for i := 0; i < elem.NumField(); i++ {
		v := elem.Field(i)
		t := elem.Type().Field(i)

		switch v.Kind() {
		case reflect.Struct:
			nv, nf := deepFields(v.Addr().Interface())
			for _, value := range nv {
				values = append(values, value)
			}
			for _, field := range nf {
				fields = append(fields, field)
			}
		default:
			values = append(values, v)
			fields = append(fields, t)
		}
	}

	return values, fields
}

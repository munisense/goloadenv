package goloadenv

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"
)

const (
	tagName = "env"
)

// EnvNotFoundError represents an error when an expected environment variable is not found.
type EnvNotFoundError struct {
	Env string
}

// Error returns a string representation of the EnvNotFoundError.
func (e *EnvNotFoundError) Error() string {
	return fmt.Sprintf("environment variable not found: %s", e.Env)
}

type EnvParseError struct {
	env   string
	err   error
	value string
}

func (e *EnvParseError) Error() string {
	return fmt.Sprintf("error parsing '%s' as environment variable %s: %s", e.value, e.env, e.err.Error())
}

// LoadEnv loads environment variables into the provided config struct.
// It uses the "env" struct tag to determine which environment variable corresponds to each field.
// If an environment variable is not found, and it does not have a default value provided in the tag, it returns an error.
//
// Example:
//
//	type DBConfig struct {
//	  Host     string `env:"DB_HOST"`
//	  Port     int    `env:"DB_PORT"`
//	  User     string `env:"DB_USER;default:user"`
//	  Password string `env:"DB_PASSWORD;default:password"`
//	}
//
//	type Config struct {
//	  Port     float64 `env:"PORT;default:8080"`
//	  LogLevel string  `env:"LOG_LEVEL;optional"`
//	  DB       DBConfig
//	}
//
//	func LoadConfig() error {
//	  cfg := Config{}
//	  err := config.LoadEnv(&cfg)
//	  if err != nil {
//	    return err
//	  }
//	  fmt.Printf("Config: %+v\n", cfg)
//	  return nil
//	}
//
//	func main() {
//	  err := godotenv.Load(".env")
//	  if err != nil {
//	    	return cfg, fmt.Errorf("could not load environment files: %w", err)
//	  }
//	  err := LoadConfig()
//	  if err != nil {
//	    fmt.Printf("Error loading config: %v\n", err)
//	    return
//	  }
//	}
//
// TODO: allow for format string defaults, function return defaults?
func LoadEnv(config interface{}) error {
	if reflect.ValueOf(config).Kind() != reflect.Ptr || reflect.ValueOf(config).Elem().Kind() != reflect.Struct {
		return errors.New("config must be a pointer to a struct")
	}
	val := reflect.ValueOf(config).Elem()
	for i := 0; i < val.NumField(); i++ {
		tags, err := getTags(val.Type().Field(i))
		if err != nil {
			return fmt.Errorf("error getting tags for field: '%s': %w", val.Type().Field(i).Name, err)
		}
		// if the field is a struct, recursively load the nested struct
		if val.Field(i).Kind() == reflect.Struct {
			err := LoadEnv(val.Field(i).Addr().Interface())
			if err != nil {
				return fmt.Errorf("error loading nested struct '%s': %w", val.Field(i).Type().Field(0).Name, err)
			}
			continue
		}
		// If field is not tagged, skip
		if tags["name"] == "" {
			continue
		}
		str, err := getField(tags)
		if err != nil {
			return err
		}
		if str == "" {
			continue
		}
		if val.Field(i).Kind() == reflect.Slice || val.Field(i).Kind() == reflect.Array {
			err = setIterableField(val.Field(i), str, tags)
			if err != nil {
				return err
			}
			continue
		}
		err = setField(val.Field(i), str, tags)
		if err != nil {
			return err
		}
	}
	return nil
}

func getTags(field reflect.StructField) (map[string]string, error) {
	unparsedTags := field.Tag.Get(tagName)
	tagSlice := strings.FieldsFunc(unparsedTags, SplitTags)
	return tagSliceToKeyMap(tagSlice)
}

// TODO support all chars in default value
// TODO allow for empty string definition of a env var, like SOMETHING=
// getField gets the value of an environment variable based on the tag. returns the value, a bool indicating if the value is optional, and an error if the value is not found.
// used internally by LoadEnv.
func getField(tags map[string]string) (string, error) {
	str := os.Getenv(tags["name"])
	if str != "" {
		return str, nil
	}
	// if the env var is not found, check if it has a default value
	if defaultValue, hasDefault := tags["default"]; hasDefault {
		return defaultValue, nil
	}
	// if the env var is not found and does not have a default value, check if it is optional
	if _, isOptional := tags["optional"]; !isOptional {
		return "", &EnvNotFoundError{Env: tags["name"]}
	}
	return "", nil
}

// setField sets the value of a field based on the string value and the field type. It returns an error if the field cannot be set or if the string value cannot be parsed into the field type.
// used internally by LoadEnv.
func setField(field reflect.Value, str string, tags map[string]string) error {
	if !field.CanSet() {
		return &EnvParseError{value: str, env: tags["name"], err: errors.New("field cannot be set")}
	}
	if unmarshaller, found := envTypes[field.Type()]; found {
		var value interface{}
		value, err := unmarshaller(str)
		if err != nil {
			return &EnvParseError{value: str, env: tags["name"], err: err}
		}
		field.Set(reflect.ValueOf(value))
	} else {
		_, err := fmt.Sscan(str, field.Addr().Interface())
		if err != nil {
			return &EnvParseError{value: str, env: tags["name"], err: err}
		}
	}
	return nil
}

// setIterableField sets the values of a field based on the string value and the underlaying iterable field type. It returns an error if the field cannot be set, if the string value cannot be parsed into the field type or if the size of the array is overflowed.
// used internally by LoadEnv.
func setIterableField(field reflect.Value, str string, tags map[string]string) error {
	if !field.CanSet() {
		return &EnvParseError{value: str, env: tags["name"], err: errors.New("field cannot be set")}
	}
	if field.Kind() != reflect.Slice && field.Kind() != reflect.Array {
		return &EnvParseError{value: str, env: tags["name"], err: errors.New("field is not a slice or array")}
	}
	maxLength := 0
	if field.Kind() == reflect.Array {
		maxLength = field.Type().Len()
	}
	strValues, err := parseArrayString(str)
	if err != nil {
		return &EnvParseError{value: str, env: tags["name"], err: err}
	}
	if maxLength > 0 && len(strValues) > maxLength {
		return &EnvParseError{value: str, env: tags["name"], err: fmt.Errorf("array size overflow, expected %d, got %d", maxLength, len(strValues))}
	}
	if field.Kind() == reflect.Slice {
		field.Set(reflect.MakeSlice(field.Type(), len(strValues), len(strValues)))
	}
	for i := 0; i < len(strValues); i++ {
		err = setField(field.Index(i), strValues[i], tags)
		if err != nil {
			return err
		}
	}
	return nil
}

func parseArrayString(str string) ([]string, error) {
	if len(str) < 2 || str[:1] != "[" && str[len(str)-1:] != "]" {
		return nil, errors.New("invalid array format")
	}
	str = str[1 : len(str)-1]
	return strings.Split(str, ","), nil
}

var tagNames = map[string]struct{}{}

// tagSliceToKeyMap converts a slice of tag strings into a map where the key is the tag and the value is the default value.
// It is used internally by LoadEnv.
func tagSliceToKeyMap(slice []string) (map[string]string, error) {
	m := make(map[string]string)
	for index := 0; index < len(slice); index++ {
		item := slice[index]
		if index == 0 {
			m["name"] = item
			if _, ok := tagNames[item]; ok {
				return nil, fmt.Errorf("duplicate tag: %s", item)
			}
			tagNames[item] = struct{}{}
			continue
		}
		if item == "default" {
			if _, ok := m[item]; ok {
				return nil, fmt.Errorf("duplicate tag: %s", item)
			}
			m[item] = slice[index+1]
			index++
			continue
		}
		m[item] = ""
	}
	return m, nil
}

// SplitTags is a helper function used to split struct tags.
// It is used internally by LoadEnv.
func SplitTags(r rune) bool {
	return r == ';' || r == ':'
}

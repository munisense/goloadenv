package goloadenv

import (
	"os"
	"strings"
	"testing"
)

type CustomMapType map[string]string

type EmbbededStruct struct {
	Host string `env:"DB_HOST;default:localhost"`
}

type EmbbededParseErrStruct struct {
	ParseErr CustomMapType `env:"PARSE_EMBEDDED_ERR;optional"`
}

type TestConfig struct {
	Host           string `env:"HOST"`
	Port           int    `env:"PORT"`
	Optional       string `env:"OPTIONAL;optional"`
	Default        string `env:"DEFAULT;default:default"`
	Struct         EmbbededStruct
	StructParseErr EmbbededParseErrStruct
	ParseErr       CustomMapType `env:"PARSE_ERR;optional"`
}

func setTestEnv() error {
	os.Clearenv()
	err := os.Setenv("HOST", "localhost")
	if err != nil {
		return err
	}
	err = os.Setenv("PORT", "8080")
	if err != nil {
		return err
	}
	err = os.Setenv("DEFAULT", "default")
	if err != nil {
		return err
	}
	return nil
}

func clearTestEnv() error {
	os.Clearenv()
	tagNames = map[string]struct{}{}
	return nil
}

func TestLoadEnv(t *testing.T) {
	clearTestEnv()

	err := os.Setenv("HOST", "localhost")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	err = os.Setenv("PORT", "8080")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	cfg := TestConfig{}
	err = LoadEnv(&cfg)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if cfg.Host != "localhost" {
		t.Errorf("Expected HOST=localhost, got %s", cfg.Host)
	}

	if cfg.Port != 8080 {
		t.Errorf("Expected PORT=8080, got %d", cfg.Port)
	}

	if cfg.Optional != "" {
		t.Errorf("Expected OPTIONAL to be empty, got %s", cfg.Optional)
	}

	if cfg.Default != "default" {
		t.Errorf("Expected DEFAULT=default, got %s", cfg.Default)
	}
}

func TestLoadEnvMissingEnv(t *testing.T) {
	clearTestEnv()

	cfg := TestConfig{}
	err := LoadEnv(&cfg)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}

	//var envNotFoundError *EnvNotFoundError
	expected := "environment variable not found: HOST"
	got := err.Error()
	if got != expected {
		t.Errorf("Expected %s, got %s", expected, got)
	}
}

func TestEnvNotFoundError(t *testing.T) {
	clearTestEnv()

	expected := "environment variable not found: HOST"
	err := LoadEnv(&TestConfig{})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if err.Error() != expected {
		t.Errorf("Expected %s, got %s", expected, err.Error())
	}
}

func TestEnvParseError(t *testing.T) {
	clearTestEnv()

	err := setTestEnv()
	if err != nil {
		t.Errorf("Error setting up test environment, got err %v", err)
	}
	err = os.Setenv("PARSE_ERR", "key1=value1,key2=value2")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	expected := "error parsing 'key1=value1,key2=value2' as environment variable PARSE_ERR: can't scan type: *load_config.CustomMapType"
	err = LoadEnv(&TestConfig{})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if err.Error() != expected {
		t.Errorf("Expected %s, got %s", expected, err.Error())
	}
}

func TestConfigStructNotAPointerError(t *testing.T) {
	clearTestEnv()

	cfg := TestConfig{}
	expected := "config must be a pointer to a struct"
	err := LoadEnv(cfg)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if err.Error() != expected {
		t.Errorf("Expected %s, got %s", expected, err.Error())
	}
}

func TestEmbeddedStructParseError(t *testing.T) {
	clearTestEnv()

	err := setTestEnv()
	if err != nil {
		t.Errorf("Error setting up test environment, got err %v", err)
	}
	err = os.Setenv("PARSE_EMBEDDED_ERR", "key1=value1,key2=value2")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expected := "error loading nested struct 'ParseErr': error parsing 'key1=value1,key2=value2' as environment variable PARSE_EMBEDDED_ERR: can't scan type: *load_config.CustomMapType"
	err = LoadEnv(&TestConfig{})
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	if err.Error() != expected {
		t.Errorf("Expected %s, got %s", expected, err.Error())
	}
}

func TestDuplicateTagNameError(t *testing.T) {
	clearTestEnv()

	err := setTestEnv()
	if err != nil {
		t.Errorf("error setting up test environment, got err %v", err)
	}

	someStruct := struct {
		Host  string `env:"HOST"`
		Host2 string `env:"HOST"`
		Host3 string `env:"HOST"`
	}{}

	err = LoadEnv(&someStruct)
	if err == nil {
		t.Errorf("Expected error, got nil")
	}
	expected := ": duplicate tag: HOST"
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("Expected %s, got %s", expected, err.Error())
	}
}

func TestSliceField(t *testing.T) {
	clearTestEnv()

	err := os.Setenv("INT_SLICE", "[1,2,3,4,5]")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	someStruct := struct {
		IntSlice []int `env:"INT_SLICE"`
	}{}

	err = LoadEnv(&someStruct)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expected := []int{1, 2, 3, 4, 5}
	if len(someStruct.IntSlice) != len(expected) {
		t.Errorf("Expected %v, got %v", expected, someStruct.IntSlice)
	}
	for i, v := range someStruct.IntSlice {
		if v != expected[i] {
			t.Errorf("Expected %v, got %v", expected, someStruct.IntSlice)
		}
	}
}

func TestArrayField(t *testing.T) {
	clearTestEnv()

	err := os.Setenv("INT_ARRAY", "[1,2,3,4,5]")
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	someStruct := struct {
		IntArray [5]int `env:"INT_ARRAY"`
	}{}

	err = LoadEnv(&someStruct)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	expected := [5]int{1, 2, 3, 4, 5}
	if someStruct.IntArray != expected {
		t.Errorf("Expected %v, got %v", expected, someStruct.IntArray)
	}
}

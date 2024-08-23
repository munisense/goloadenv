package goloadenv

import (
	"log/slog"
	"reflect"
)

type EnvType func(string) (interface{}, error)

type EnvTypeInterface interface {
	UnmarshalEnv(string) (interface{}, error)
}

var envTypes = map[reflect.Type]EnvType{
	reflect.TypeFor[slog.Level](): UnmarshalEnvSlogLevel,
}

func RegisterEnvType[T EnvTypeInterface]() {
	var proto T
	envTypes[reflect.TypeFor[T]()] = proto.UnmarshalEnv
}

func UnmarshalEnvSlogLevel(string string) (interface{}, error) {
	var level slog.Level
	return level, level.UnmarshalText([]byte(string))
}

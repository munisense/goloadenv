package goloadenv

import (
	"fmt"
	"reflect"
	"strings"
)

// TODO maybe an extra tag that ensures a field is not printed, handy for passwords for example
func FormatString(config interface{}) string {
	return fmt.Sprintf("{\n%s\n}", formatStruct(reflect.ValueOf(config), 1))
}

func formatStruct(v reflect.Value, indent int) string {
	if v.Kind() == reflect.Ptr {
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return fmt.Sprint(v.Interface())
	}

	var lines []string
	maxLen := getMaxFieldNameLength(v)

	for i := 0; i < v.NumField(); i++ {
		fieldType := v.Type().Field(i)
		fieldValue := v.Field(i)
		indentation := strings.Repeat("    ", indent)

		if fieldValue.Kind() == reflect.Struct {
			lines = append(lines, fmt.Sprintf("%s%-*s {\n%s\n%s}", indentation, maxLen, fmt.Sprintf("%s:", fieldType.Name), formatStruct(fieldValue, indent+1), indentation))
		} else {
			lines = append(lines, fmt.Sprintf("%s%-*s %v", indentation, maxLen, fmt.Sprintf("%s:", fieldType.Name), fieldValue.Interface()))
		}
	}

	return strings.Join(lines, "\n")
}

func getMaxFieldNameLength(v reflect.Value) int {
	maxLen := 0
	for i := 0; i < v.NumField(); i++ {
		fieldType := v.Type().Field(i)
		if len(fieldType.Name)+1 > maxLen {
			maxLen = len(fieldType.Name) + 1
		}
	}
	return maxLen
}

package env

import (
	"os"
	"strconv"
)

func GetEnv(key string, defaultValue string) string {
	value, ok := os.LookupEnv(key)
	if ok {
		return value
	}
	return defaultValue
}

func GetInt(key string, defaultValue int) int {
	value, ok := os.LookupEnv(key)
	if ok {
		valueInt, err := strconv.Atoi(value)
		if err != nil {
			panic(err) // expected developer provide suitable config
		}
		return valueInt
	}
	return defaultValue
}

func GetBool(key string, defaultValue bool) bool {
	value, ok := os.LookupEnv(key)
	if ok {
		return (value == "true")
	}
	return defaultValue
}

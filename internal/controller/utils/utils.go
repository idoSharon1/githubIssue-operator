package utils

import (
	"os"
)

func SetEnvironmentVariable(key string, value string) error {
	err := os.Setenv(key, value)

	if err != nil {
		return err
	} else {
		return nil
	}
}

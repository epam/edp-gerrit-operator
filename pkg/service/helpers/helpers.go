package helpers

import (
	"log"
)

// LogErrorAndReturn prints error message to the log and returns err parameter
func LogErrorAndReturn(err error) error {
	log.Printf("[ERROR] %v", err)
	return err
}

// GenerateLabels returns initialized map using name parameter
func GenerateLabels(name string) map[string]string {
	return map[string]string{
		"app": name,
	}
}

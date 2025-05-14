package utils

import "fmt"

// MapToEnvList converts a map to a slice of "key=value" strings.
func MapToEnvList(env map[string]string) []string {
	envList := make([]string, 0, len(env))
	for k, v := range env {
		envList = append(envList, fmt.Sprintf("%s=%s", k, v))
	}
	return envList
}

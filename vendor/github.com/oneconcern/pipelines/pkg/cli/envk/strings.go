package envk

import (
	"os"
	"strings"
)

// StringOrDefault environment value for the specified key
func StringOrDefault(key, defVal string) string {
	v := os.Getenv(key)
	if v == "" {
		return defVal
	}
	return v
}

// StringSliceOrDefault environment value for the specified key, splits on , or the provided delimiter
func StringSliceOrDefault(key, delimiter string, defVal []string) []string {
	if delimiter == "" {
		delimiter = ","
	}

	v := os.Getenv(key)
	if v == "" {
		return defVal
	}

	var result []string
	for _, v := range strings.Split(v, delimiter) {
		r := strings.TrimSpace(v)
		if r == "" {
			continue
		}
		result = append(result, r)
	}
	return result
}

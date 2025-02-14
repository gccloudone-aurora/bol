package util

import (
	"log"
	"net/url"
)

// MustParseURL parses a URL and panics if it fails.
func MustParseURL(urlStr string) *url.URL {
	url, err := url.Parse(urlStr)
	if err != nil {
		log.Fatalf("failed to parse url: %v", err)
	}

	return url
}

// GetOrDefault returns the value for a key if present, otherwise returns the default value.
func GetOrDefault[K comparable, V any](m map[K]V, key K, defaultValue V) V {
	if value, ok := m[key]; ok {
		return value
	}
	return defaultValue
}

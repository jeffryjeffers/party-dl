package utils

import "strings"

var (
	supportedURLs = []string{"https://coomer.su"}
)

func IsURlSupported(url string) bool {
	for _, supportedURL := range supportedURLs {
		if strings.HasPrefix(url, supportedURL) {
			return true
		}
	}
	return false
}

func GetService(url string) string {
	if strings.HasPrefix(url, "https://onlyfans.com") {
		return "onlyfans"
	} else if strings.HasPrefix(url, "https://fansly.com") {
		return "fansly"
	}
	return "unknown"
}

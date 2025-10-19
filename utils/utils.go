package utils

import "net/http"

func GetUrl(r *http.Request, path string) string {
	query := r.URL.RawQuery
	if query != "" {
		return path + "?" + query
	}
	return path
}

package ciauth

import (
	"fmt"
	"net/http"
	"strings"
)

type Config struct {
	Providers map[string]string
}

func extractBearerToken(header http.Header) (string, error) {
	reqToken := header.Get("Authorization")
	splitToken := strings.Split(reqToken, "Bearer")
	if len(splitToken) != 2 {
		return "", fmt.Errorf("malformed token")
	}

	return strings.TrimSpace(splitToken[1]), nil
}

package utils

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
)

const certPath = "/opt/nautes/cert"

func GetCABundle(serviceURL string) ([]byte, error) {
	u, err := url.Parse(serviceURL)
	if err != nil {
		return nil, fmt.Errorf("parse url failed: %w", err)
	}
	fileName := fmt.Sprintf("%s_%s", u.Hostname(), u.Port())
	fileFullPath := filepath.Join(certPath, fileName)
	return os.ReadFile(fileFullPath)
}

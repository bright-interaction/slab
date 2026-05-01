//go:build !ee

package config

import (
	"strings"
	"testing"
)

// TestValidate_DeploymentMode_OSS confirms that on the OSS build,
// ATOMICSITE_DEPLOYMENT_MODE=cloud is refused with a message that
// directs the operator to rebuild with -tags ee.
func TestValidate_DeploymentMode_OSS(t *testing.T) {
	for _, baseURL := range []string{"http://localhost:8080", "https://atomicsite.example.com"} {
		t.Run(baseURL, func(t *testing.T) {
			cfg := goodConfig()
			cfg.BaseURL = baseURL
			cfg.DeploymentMode = DeploymentModeCloud
			err := cfg.Validate()
			if err == nil {
				t.Fatal("expected cloud-mode rejection on OSS build")
			}
			if !strings.Contains(err.Error(), "requires a binary built with -tags ee") {
				t.Errorf("error = %q, want substring %q", err.Error(), "requires a binary built with -tags ee")
			}
		})
	}
}

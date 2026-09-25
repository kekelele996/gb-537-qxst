package util

import (
	"strings"
	"testing"
)

func TestRedactedJSONRemovesSecretsAndCertificateBodies(t *testing.T) {
	raw := RedactedJSON(map[string]any{"password": "secret", "certificate_pem": "public body", "nested": map[string]any{"token": "jwt", "safe": "fingerprint"}})
	for _, secret := range []string{"secret", "public body", "jwt"} {
		if strings.Contains(raw, secret) {
			t.Fatalf("secret %q leaked in %s", secret, raw)
		}
	}
	if !strings.Contains(raw, "fingerprint") || strings.Count(raw, "[REDACTED]") != 3 {
		t.Fatalf("unexpected redaction: %s", raw)
	}
}

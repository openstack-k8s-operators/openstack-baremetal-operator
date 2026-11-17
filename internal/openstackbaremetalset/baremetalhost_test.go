package openstackbaremetalset

import (
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
)

func TestSplitPEMCerts(t *testing.T) {
	bundle := `-----BEGIN CERTIFICATE-----
CERTONE
-----END CERTIFICATE-----
-----BEGIN CERTIFICATE-----
CERTTWO
-----END CERTIFICATE-----
`
	certs := splitPEMCerts(bundle)
	if len(certs) != 2 {
		t.Fatalf("expected 2 certs, got %d", len(certs))
	}
	if !strings.Contains(certs[0], "CERTONE") {
		t.Fatalf("first cert missing CERTONE: %q", certs[0])
	}
	if !strings.Contains(certs[1], "CERTTWO") {
		t.Fatalf("second cert missing CERTTWO: %q", certs[1])
	}
}

func TestTrustedCACertsFromSecret(t *testing.T) {
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"z.crt": []byte(`-----BEGIN CERTIFICATE-----
ZCERT
-----END CERTIFICATE-----`),
			"a.crt": []byte(`-----BEGIN CERTIFICATE-----
ACERT
-----END CERTIFICATE-----`),
		},
	}

	certs := trustedCACertsFromSecret(secret)
	if len(certs) != 2 {
		t.Fatalf("expected 2 certs, got %d", len(certs))
	}
	// Keys are sorted, so a.crt comes before z.crt
	if !strings.HasPrefix(certs[0], "    -----BEGIN CERTIFICATE-----") {
		t.Fatalf("cert should be indented with 4 spaces: %q", certs[0])
	}
	if !strings.Contains(certs[0], "ACERT") {
		t.Fatalf("expected ACERT first due to key sort, got %q", certs[0])
	}
	if !strings.Contains(certs[1], "ZCERT") {
		t.Fatalf("expected ZCERT second, got %q", certs[1])
	}
}

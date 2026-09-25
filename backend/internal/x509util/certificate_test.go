package x509util

import (
	"strings"
	"testing"
	"time"
)

func TestGeneratedPublicChainVerifiesWithStandardLibrary(t *testing.T) {
	now := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	fixture, err := GenerateDemoPKI(now)
	if err != nil {
		t.Fatal(err)
	}
	evidence, refs, err := ValidateChain(fixture.OldRootPEM, fixture.OldLeafPEM, now)
	if err != nil {
		t.Fatal(err)
	}
	if !evidence.Valid || len(evidence.PathSubjects) != 2 {
		t.Fatalf("unexpected evidence: %+v", evidence)
	}
	if len(refs) != 1 || !strings.Contains(refs[0].Subject, "api.platform.example.internal") {
		t.Fatalf("unexpected refs: %+v", refs)
	}
}

func TestPrivateKeyMaterialIsRejected(t *testing.T) {
	privatePEM := "-----BEGIN PRIVATE KEY-----\nZm9yYmlkZGVu\n-----END PRIVATE KEY-----"
	if _, err := ParseCertificatePEM(privatePEM); err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("expected private key rejection, got %v", err)
	}
	if _, err := ParseCertificateBundle(privatePEM); err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("expected bundle private key rejection, got %v", err)
	}
}

func TestWrongRootCannotValidateLeaf(t *testing.T) {
	now := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	fixture, err := GenerateDemoPKI(now)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := ValidateChain(fixture.NewRootPEM, fixture.OldLeafPEM, now); err == nil {
		t.Fatal("leaf signed by old root must not validate under new root")
	}
}

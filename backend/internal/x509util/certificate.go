package x509util

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"
)

type CertificateRef struct {
	Subject           string    `json:"subject"`
	Issuer            string    `json:"issuer"`
	SerialNumber      string    `json:"serial_number"`
	FingerprintSHA256 string    `json:"fingerprint_sha256"`
	NotBefore         time.Time `json:"not_before"`
	NotAfter          time.Time `json:"not_after"`
	IsCA              bool      `json:"is_ca"`
}

type ChainEvidence struct {
	Valid        bool      `json:"valid"`
	VerifiedAt   time.Time `json:"verified_at"`
	LeafSubject  string    `json:"leaf_subject"`
	RootSubject  string    `json:"root_subject"`
	PathSubjects []string  `json:"path_subjects"`
	Message      string    `json:"message"`
}

func ParseCertificatePEM(raw string) (*x509.Certificate, error) {
	upper := strings.ToUpper(raw)
	if strings.Contains(upper, "PRIVATE KEY") {
		return nil, fmt.Errorf("private key material is forbidden")
	}
	block, rest := pem.Decode([]byte(strings.TrimSpace(raw)))
	if block == nil || block.Type != "CERTIFICATE" {
		return nil, fmt.Errorf("input must contain a PEM CERTIFICATE block")
	}
	if len(strings.TrimSpace(string(rest))) != 0 {
		return nil, fmt.Errorf("single certificate input contains trailing material")
	}
	certificate, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse X.509 certificate: %w", err)
	}
	return certificate, nil
}

func ParseCertificateBundle(raw string) ([]*x509.Certificate, error) {
	upper := strings.ToUpper(raw)
	if strings.Contains(upper, "PRIVATE KEY") {
		return nil, fmt.Errorf("private key material is forbidden")
	}
	remaining := []byte(strings.TrimSpace(raw))
	certificates := make([]*x509.Certificate, 0, 4)
	for len(strings.TrimSpace(string(remaining))) > 0 {
		block, rest := pem.Decode(remaining)
		if block == nil || block.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("bundle contains non-certificate or malformed PEM material")
		}
		certificate, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse bundle certificate: %w", err)
		}
		certificates = append(certificates, certificate)
		remaining = rest
	}
	if len(certificates) == 0 {
		return nil, fmt.Errorf("certificate bundle is empty")
	}
	return certificates, nil
}

func Fingerprint(certificate *x509.Certificate) string {
	sum := sha256.Sum256(certificate.Raw)
	return hex.EncodeToString(sum[:])
}
func NormalizePEM(certificate *x509.Certificate) string {
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certificate.Raw}))
}
func SubjectDN(certificate *x509.Certificate) string { return certificate.Subject.String() }
func KeyAlgorithm(certificate *x509.Certificate) string {
	return certificate.PublicKeyAlgorithm.String()
}
func Ref(certificate *x509.Certificate) CertificateRef {
	return CertificateRef{Subject: certificate.Subject.String(), Issuer: certificate.Issuer.String(), SerialNumber: certificate.SerialNumber.String(), FingerprintSHA256: Fingerprint(certificate), NotBefore: certificate.NotBefore.UTC(), NotAfter: certificate.NotAfter.UTC(), IsCA: certificate.IsCA}
}

func ValidateTrustAnchor(certificate *x509.Certificate, at time.Time) error {
	if !certificate.IsCA || !certificate.BasicConstraintsValid {
		return fmt.Errorf("trust anchor must be a CA certificate")
	}
	if certificate.KeyUsage&x509.KeyUsageCertSign == 0 {
		return fmt.Errorf("trust anchor is missing certificate-signing key usage")
	}
	if at.Before(certificate.NotBefore) || !at.Before(certificate.NotAfter) {
		return fmt.Errorf("trust anchor is not valid at %s", at.UTC().Format(time.RFC3339))
	}
	if err := certificate.CheckSignatureFrom(certificate); err != nil {
		return fmt.Errorf("trust anchor must be self-signed: %w", err)
	}
	return nil
}

func ValidateChain(anchorPEM, bundlePEM string, at time.Time) (ChainEvidence, []CertificateRef, error) {
	anchor, err := ParseCertificatePEM(anchorPEM)
	if err != nil {
		return ChainEvidence{}, nil, fmt.Errorf("parse trust anchor: %w", err)
	}
	if err := ValidateTrustAnchor(anchor, at); err != nil {
		return ChainEvidence{}, nil, err
	}
	certificates, err := ParseCertificateBundle(bundlePEM)
	if err != nil {
		return ChainEvidence{}, nil, err
	}
	leaf := certificates[0]
	roots := x509.NewCertPool()
	roots.AddCert(anchor)
	intermediates := x509.NewCertPool()
	for _, certificate := range certificates[1:] {
		intermediates.AddCert(certificate)
	}
	chains, err := leaf.Verify(x509.VerifyOptions{Roots: roots, Intermediates: intermediates, CurrentTime: at, KeyUsages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}})
	if err != nil {
		return ChainEvidence{}, nil, fmt.Errorf("verify public certificate chain: %w", err)
	}
	path := make([]string, 0, len(chains[0]))
	for _, certificate := range chains[0] {
		path = append(path, certificate.Subject.String())
	}
	refs := make([]CertificateRef, 0, len(certificates))
	for _, certificate := range certificates {
		refs = append(refs, Ref(certificate))
	}
	sort.Slice(refs, func(i, j int) bool { return refs[i].Subject < refs[j].Subject })
	return ChainEvidence{Valid: true, VerifiedAt: at.UTC(), LeafSubject: leaf.Subject.String(), RootSubject: anchor.Subject.String(), PathSubjects: path, Message: "signature path and validity window verified offline"}, refs, nil
}

type DemoPKI struct{ OldRootPEM, NewRootPEM, OldLeafPEM, NewLeafPEM string }

func GenerateDemoPKI(now time.Time) (DemoPKI, error) {
	oldRoot, oldRootKey, oldRootPEM, err := createRoot("Example Manufacturing Legacy Root", 1001, now.AddDate(-3, 0, 0), now.AddDate(0, 6, 0))
	if err != nil {
		return DemoPKI{}, err
	}
	newRoot, newRootKey, newRootPEM, err := createRoot("Example Manufacturing 2031 Root", 2001, now.AddDate(0, -1, 0), now.AddDate(6, 0, 0))
	if err != nil {
		return DemoPKI{}, err
	}
	oldLeafPEM, err := createLeaf("api.platform.example.internal", 1101, now.AddDate(-1, 0, 0), now.AddDate(0, 4, 0), oldRoot, oldRootKey)
	if err != nil {
		return DemoPKI{}, err
	}
	newLeafPEM, err := createLeaf("api.platform.example.internal", 2101, now.AddDate(0, 0, -1), now.AddDate(2, 0, 0), newRoot, newRootKey)
	if err != nil {
		return DemoPKI{}, err
	}
	return DemoPKI{OldRootPEM: oldRootPEM, NewRootPEM: newRootPEM, OldLeafPEM: oldLeafPEM, NewLeafPEM: newLeafPEM}, nil
}

func createRoot(commonName string, serial int64, notBefore, notAfter time.Time) (*x509.Certificate, *ecdsa.PrivateKey, string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, "", fmt.Errorf("generate root key: %w", err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{Organization: []string{"Example Manufacturing"}, CommonName: commonName}, NotBefore: notBefore.UTC(), NotAfter: notAfter.UTC(), IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature, SignatureAlgorithm: x509.ECDSAWithSHA256}
	raw, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		return nil, nil, "", fmt.Errorf("create root certificate: %w", err)
	}
	certificate, err := x509.ParseCertificate(raw)
	if err != nil {
		return nil, nil, "", err
	}
	return certificate, key, string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: raw})), nil
}

func createLeaf(commonName string, serial int64, notBefore, notAfter time.Time, issuer *x509.Certificate, issuerKey *ecdsa.PrivateKey) (string, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return "", fmt.Errorf("generate leaf key: %w", err)
	}
	template := &x509.Certificate{SerialNumber: big.NewInt(serial), Subject: pkix.Name{Organization: []string{"Example Manufacturing Platform"}, CommonName: commonName}, DNSNames: []string{commonName}, NotBefore: notBefore.UTC(), NotAfter: notAfter.UTC(), BasicConstraintsValid: true, KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}, SignatureAlgorithm: x509.ECDSAWithSHA256}
	raw, err := x509.CreateCertificate(rand.Reader, template, issuer, &key.PublicKey, issuerKey)
	if err != nil {
		return "", fmt.Errorf("create leaf certificate: %w", err)
	}
	return string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: raw})), nil
}

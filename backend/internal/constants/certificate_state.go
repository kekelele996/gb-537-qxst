package constants

import "time"

type CertificateState string

const (
	CertificateValid    CertificateState = "valid"
	CertificateExpiring CertificateState = "expiring"
	CertificateExpired  CertificateState = "expired"
	CertificateRevoked  CertificateState = "revoked"
)

func CertificateStateValues() []CertificateState {
	return []CertificateState{CertificateValid, CertificateExpiring, CertificateExpired, CertificateRevoked}
}

func CalculateCertificateState(now, notAfter time.Time, revoked bool) CertificateState {
	if revoked {
		return CertificateRevoked
	}
	if !now.Before(notAfter) {
		return CertificateExpired
	}
	if !now.Add(90 * 24 * time.Hour).Before(notAfter) {
		return CertificateExpiring
	}
	return CertificateValid
}

type ChainState string

const (
	ChainImported   ChainState = "imported"
	ChainValidated  ChainState = "validated"
	ChainDeprecated ChainState = "deprecated"
	ChainRevoked    ChainState = "revoked"
)

func CanTransitionChain(from, to ChainState) bool {
	allowed := map[ChainState][]ChainState{
		ChainImported:   {ChainValidated, ChainRevoked},
		ChainValidated:  {ChainDeprecated, ChainRevoked},
		ChainDeprecated: {ChainRevoked},
	}
	for _, candidate := range allowed[from] {
		if candidate == to {
			return true
		}
	}
	return false
}

type ServiceState string

const (
	ServiceActive   ServiceState = "active"
	ServiceInactive ServiceState = "inactive"
)

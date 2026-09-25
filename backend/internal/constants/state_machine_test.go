package constants

import (
	"testing"
	"time"
)

func TestCertificateStateIsDerivedFromEvidence(t *testing.T) {
	now := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	cases := []struct {
		name     string
		notAfter time.Time
		revoked  bool
		want     CertificateState
	}{{"valid", now.Add(180 * 24 * time.Hour), false, CertificateValid}, {"expiring", now.Add(30 * 24 * time.Hour), false, CertificateExpiring}, {"expired", now.Add(-time.Second), false, CertificateExpired}, {"revoked wins", now.Add(365 * 24 * time.Hour), true, CertificateRevoked}}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if got := CalculateCertificateState(now, test.notAfter, test.revoked); got != test.want {
				t.Fatalf("got %s want %s", got, test.want)
			}
		})
	}
}

func TestScenarioStateMachine(t *testing.T) {
	legal := [][2]ScenarioState{{ScenarioDraft, ScenarioSimulated}, {ScenarioSimulated, ScenarioReady}, {ScenarioSimulated, ScenarioDraft}, {ScenarioReady, ScenarioExecuting}, {ScenarioReady, ScenarioDraft}, {ScenarioExecuting, ScenarioVerified}, {ScenarioExecuting, ScenarioRollback}}
	for _, pair := range legal {
		if !CanTransitionScenario(pair[0], pair[1]) {
			t.Fatalf("expected legal %s -> %s", pair[0], pair[1])
		}
	}
	illegal := [][2]ScenarioState{{ScenarioDraft, ScenarioVerified}, {ScenarioSimulated, ScenarioExecuting}, {ScenarioReady, ScenarioVerified}, {ScenarioVerified, ScenarioDraft}, {ScenarioRollback, ScenarioExecuting}}
	for _, pair := range illegal {
		if CanTransitionScenario(pair[0], pair[1]) {
			t.Fatalf("expected illegal %s -> %s", pair[0], pair[1])
		}
	}
}

func TestRBACLeastPrivilege(t *testing.T) {
	if !HasPermission(RolePKIOperator, PermissionAnchorWrite) {
		t.Fatal("PKI operator should import anchors")
	}
	if HasPermission(RoleAuditor, PermissionAnchorWrite) {
		t.Fatal("auditor must not write anchors")
	}
	if !HasPermission(RoleServiceOwner, PermissionDependencyWrite) {
		t.Fatal("service owner should maintain owned dependencies")
	}
	if HasPermission(RoleServiceOwner, PermissionScenarioVerify) {
		t.Fatal("service owner must not verify scenarios")
	}
	if !HasPermission(RoleSecurityReviewer, PermissionScenarioVerify) {
		t.Fatal("reviewer should verify independent scenarios")
	}
}

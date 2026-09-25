package constants

type Role string
type Permission string

const (
	RoleAdmin            Role = "admin"
	RolePKIOperator      Role = "pki_operator"
	RoleServiceOwner     Role = "service_owner"
	RoleSecurityReviewer Role = "security_reviewer"
	RoleAuditor          Role = "auditor"

	PermissionRead            Permission = "read"
	PermissionAnchorWrite     Permission = "anchor.write"
	PermissionChainWrite      Permission = "chain.write"
	PermissionDependencyWrite Permission = "dependency.write"
	PermissionScenarioWrite   Permission = "scenario.write"
	PermissionScenarioRun     Permission = "scenario.run"
	PermissionScenarioVerify  Permission = "scenario.verify"
	PermissionAuditRead       Permission = "audit.read"
)

var permissions = map[Role]map[Permission]bool{
	RoleAdmin:            {PermissionRead: true, PermissionAnchorWrite: true, PermissionChainWrite: true, PermissionDependencyWrite: true, PermissionScenarioWrite: true, PermissionScenarioRun: true, PermissionScenarioVerify: true, PermissionAuditRead: true},
	RolePKIOperator:      {PermissionRead: true, PermissionAnchorWrite: true, PermissionChainWrite: true, PermissionDependencyWrite: true, PermissionScenarioWrite: true, PermissionScenarioRun: true},
	RoleServiceOwner:     {PermissionRead: true, PermissionDependencyWrite: true, PermissionScenarioWrite: true, PermissionScenarioRun: true},
	RoleSecurityReviewer: {PermissionRead: true, PermissionScenarioVerify: true, PermissionAuditRead: true},
	RoleAuditor:          {PermissionRead: true, PermissionAuditRead: true},
}

func HasPermission(role Role, permission Permission) bool { return permissions[role][permission] }
func RoleValues() []Role {
	return []Role{RoleAdmin, RolePKIOperator, RoleServiceOwner, RoleSecurityReviewer, RoleAuditor}
}

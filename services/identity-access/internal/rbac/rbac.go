package rbac

import "identity-access/internal/auth"

type Evaluator struct {
	rolePerms map[string][]string
}

func NewEvaluator() *Evaluator {
	return &Evaluator{
		rolePerms: map[string][]string {
			"platform_admin": {"*"},
			"tenant_admin": {
				"case:read", "case:write",
				"workflow:read", "workflow:write",
				"decision:read",
			},
			"analyst": {
				"case:read", "case:write",
				"decision:read",
			},
			"merchant_user": {
				"transaction:read",
				"decision:read",
			},
			"ops_admin": {
				"ops:read", 
				"audit:read",
			},
		},
}
}

func (e *Evaluator) EffectivePermissions(p auth.Principal) []string {
	set := map[string]struct{}{}
	for _, role := range p.Roles {
		for _, perm := range e.rolePerms[role] {
			set[perm] = struct{}{}
		}
	}

	out := make([]string, 0, len(set))

	for perm := range set {
		out = append(out, perm)
	}
	return out
}

func (e *Evaluator) Can(p auth.Principal, action, resourceTenantID string) (allowed bool, reason string, effective []string) {
	effectivePerms := e.EffectivePermissions(p)

	if resourceTenantID != "" && resourceTenantID != p.TenantID && !hasRole(p.Roles, "platform_admin") {
		return false,"tenant_mismatch", effectivePerms
	}

	if hasPerm(effectivePerms, "*") || hasPerm(effectivePerms, action) {
		return true, "allowed", effectivePerms
	}
	return false, "permission_denied", effectivePerms
}

func hasRole(roles []string, role string) bool {
	for _, r := range roles {
		if r == role {
			return true
		}
	}
	return false
}

func hasPerm(perms []string, perm string) bool {
	for _, p := range perms {
		if p == perm {
			return true
		}
	}
	return false
}
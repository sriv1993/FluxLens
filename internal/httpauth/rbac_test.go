package httpauth

import "testing"

func TestParseKeyRoles(t *testing.T) {
	m := ParseKeyRoles([]string{"k1:operator+admin", "k2", "k3:auditor"})
	if len(m["k1"]) != 2 || m["k1"][0] != RoleOperator {
		t.Fatalf("k1 roles: %v", m["k1"])
	}
	if len(m["k2"]) != 1 || m["k2"][0] != RoleOperator {
		t.Fatalf("k2 default: %v", m["k2"])
	}
	if m["k3"][0] != RoleAuditor {
		t.Fatalf("k3: %v", m["k3"])
	}
}

package teams

import "testing"

func ms(role Role) Membership {
	return Membership{Role: role, Status: MembershipStatusActive}
}

func TestCanApproveProof(t *testing.T) {
	cases := []struct {
		name string
		mem  Membership
		want bool
	}{
		{"lead can approve", ms(RoleLead), true},
		{"trusted can approve", ms(RoleTrustedApprover), true},
		{"member cannot approve", ms(RoleMember), false},
		{"left lead cannot approve", Membership{Role: RoleLead, Status: MembershipStatusLeft}, false},
		{"removed trusted cannot approve", Membership{Role: RoleTrustedApprover, Status: MembershipStatusRemoved}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := CanApproveProof(c.mem); got != c.want {
				t.Errorf("CanApproveProof(%v) = %v, want %v", c.mem, got, c.want)
			}
		})
	}
}

func TestCanApproveProofForOwnerRejectsSelf(t *testing.T) {
	mem := Membership{UserID: 42, Role: RoleLead, Status: MembershipStatusActive}
	if CanApproveProofForOwner(mem, 42) {
		t.Error("CanApproveProofForOwner(self) = true, want false (cannot approve own)")
	}
	if !CanApproveProofForOwner(mem, 7) {
		t.Error("CanApproveProofForOwner(other) = false, want true")
	}
}

func TestCanManageTeam(t *testing.T) {
	if !CanManageTeam(ms(RoleLead)) {
		t.Error("lead should be able to manage team")
	}
	if CanManageTeam(ms(RoleTrustedApprover)) {
		t.Error("trusted should NOT manage team")
	}
	if CanManageTeam(ms(RoleMember)) {
		t.Error("member should NOT manage team")
	}
	if CanManageTeam(Membership{Role: RoleLead, Status: MembershipStatusLeft}) {
		t.Error("left lead should NOT manage team")
	}
}

func TestCanReadFullMemberMetrics(t *testing.T) {
	if !CanReadFullMemberMetrics(ms(RoleLead)) {
		t.Error("lead should read full member metrics")
	}
	if !CanReadFullMemberMetrics(ms(RoleTrustedApprover)) {
		t.Error("trusted should read full member metrics")
	}
	if CanReadFullMemberMetrics(ms(RoleMember)) {
		t.Error("member should NOT read others' full metrics")
	}
}

func TestCanReadPeerMetrics(t *testing.T) {
	for _, r := range []Role{RoleLead, RoleTrustedApprover, RoleMember} {
		if !CanReadPeerMetrics(ms(r)) {
			t.Errorf("active %v should read peer metrics", r)
		}
	}
	if CanReadPeerMetrics(Membership{Role: RoleMember, Status: MembershipStatusLeft}) {
		t.Error("left member should NOT read peer metrics")
	}
}

func TestEnsureCanChangeAIConsentOnlySelf(t *testing.T) {
	if err := EnsureCanChangeAIConsent(7, 7); err != nil {
		t.Errorf("self should be allowed, got %v", err)
	}
	if err := EnsureCanChangeAIConsent(7, 8); err == nil {
		t.Error("changing other's consent should fail")
	}
}

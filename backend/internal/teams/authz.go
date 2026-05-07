package teams

// authz.go — pure authorization functions. Every call site that gates an
// action by role MUST go through one of these helpers. Inlining
// `if mem.Role == RoleLead` in handlers/services is forbidden because it
// hides the policy and makes drift between surfaces inevitable.
//
// All functions take a Membership by value (small struct; no pointer
// indirection needed) and return either a bool or an error from the
// domain.go list. They never read the database.

// CanApproveProof returns true iff the membership has approval rights and
// is active. Rights:
//
//	lead             — yes
//	trusted_approver — yes
//	member           — no
//
// A non-active membership (left/removed) is always false regardless of role.
func CanApproveProof(mem Membership) bool {
	if mem.Status != MembershipStatusActive {
		return false
	}
	return mem.Role == RoleLead || mem.Role == RoleTrustedApprover
}

// CanApproveProofForOwner additionally rejects approving one's own proof.
// Caller must pass the owner_user_id of the proof being approved.
func CanApproveProofForOwner(approver Membership, ownerUserID int64) bool {
	if approver.UserID == ownerUserID {
		return false
	}
	return CanApproveProof(approver)
}

// CanManageTeam — only the lead can rename, archive, change AI mode,
// regenerate invite, change other members' roles, remove members.
func CanManageTeam(mem Membership) bool {
	return mem.Status == MembershipStatusActive && mem.Role == RoleLead
}

// CanReadFullMemberMetrics — lead and trusted may read /members/:userId/full.
// Member can only read /full for self (handled by EnsureSelf in the handler).
func CanReadFullMemberMetrics(mem Membership) bool {
	if mem.Status != MembershipStatusActive {
		return false
	}
	return mem.Role == RoleLead || mem.Role == RoleTrustedApprover
}

// CanReadPeerMetrics — every active member of the team may see peer cards.
func CanReadPeerMetrics(mem Membership) bool {
	return mem.Status == MembershipStatusActive
}

// EnsureCanChangeAIConsent — only the user themselves can flip their consent.
// Returns ErrCannotChangeOthersAIConsent on mismatch.
func EnsureCanChangeAIConsent(callerUserID, targetUserID int64) error {
	if callerUserID == targetUserID {
		return nil
	}
	return ErrCannotChangeOthersAIConsent
}

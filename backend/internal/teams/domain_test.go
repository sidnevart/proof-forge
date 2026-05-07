package teams

import (
	"strings"
	"testing"
)

func TestRoleString(t *testing.T) {
	cases := []struct {
		role Role
		want string
	}{
		{RoleLead, "lead"},
		{RoleTrustedApprover, "trusted_approver"},
		{RoleMember, "member"},
	}
	for _, c := range cases {
		if string(c.role) != c.want {
			t.Errorf("Role %v string = %q, want %q", c.role, string(c.role), c.want)
		}
	}
}

func TestParseRoleAcceptsValidValues(t *testing.T) {
	cases := []struct {
		in   string
		want Role
	}{
		{"lead", RoleLead},
		{"trusted_approver", RoleTrustedApprover},
		{"member", RoleMember},
	}
	for _, c := range cases {
		got, err := ParseRole(c.in)
		if err != nil {
			t.Fatalf("ParseRole(%q) returned error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("ParseRole(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseRoleRejectsUnknown(t *testing.T) {
	if _, err := ParseRole("admin"); err == nil {
		t.Fatal(`ParseRole("admin") expected error, got nil`)
	}
	if _, err := ParseRole(""); err == nil {
		t.Fatal(`ParseRole("") expected error, got nil`)
	}
}

func TestParseAIModeAcceptsValidValues(t *testing.T) {
	cases := []struct {
		in   string
		want AIMode
	}{
		{"off", AIModeOff},
		{"metadata-only", AIModeMetadataOnly},
		{"full", AIModeFull},
	}
	for _, c := range cases {
		got, err := ParseAIMode(c.in)
		if err != nil {
			t.Fatalf("ParseAIMode(%q) returned error: %v", c.in, err)
		}
		if got != c.want {
			t.Errorf("ParseAIMode(%q) = %v, want %v", c.in, got, c.want)
		}
	}
}

func TestParseAIModeRejectsUnknown(t *testing.T) {
	if _, err := ParseAIMode("hyper"); err == nil {
		t.Fatal(`ParseAIMode("hyper") expected error, got nil`)
	}
}

func TestValidateTeamNameRejectsEmpty(t *testing.T) {
	if err := ValidateTeamName(""); err == nil {
		t.Fatal(`ValidateTeamName("") expected error`)
	}
	if err := ValidateTeamName("   "); err == nil {
		t.Fatal("ValidateTeamName whitespace expected error")
	}
}

func TestValidateTeamNameRejectsTooLong(t *testing.T) {
	long := strings.Repeat("я", 81)
	if err := ValidateTeamName(long); err == nil {
		t.Fatal("ValidateTeamName(81 chars) expected error")
	}
}

func TestValidateTeamNameAcceptsBoundary(t *testing.T) {
	if err := ValidateTeamName("a"); err != nil {
		t.Fatalf("ValidateTeamName(1 char) unexpected error: %v", err)
	}
	long := strings.Repeat("я", 80)
	if err := ValidateTeamName(long); err != nil {
		t.Fatalf("ValidateTeamName(80 chars) unexpected error: %v", err)
	}
}

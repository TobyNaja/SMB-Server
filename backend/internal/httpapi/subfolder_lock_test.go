package httpapi

import (
	"strings"
	"testing"
)

// buildLockCmd must, in order: wipe the ACL, shut out "other", then grant the
// allowlist — and give each user traverse (x) up to the share root.
func TestBuildLockCmd_ShutsOutOtherThenGrantsAllowlist(t *testing.T) {
	cmd := buildLockCmd("", "/mnt/shared/s", "/mnt/shared/s/Secret", "rx", []string{"toby"})

	iBlank := strings.Index(cmd, "setfacl -b '/mnt/shared/s/Secret'")
	iChmod := strings.Index(cmd, "chmod o= '/mnt/shared/s/Secret'")
	iAcc := strings.Index(cmd, "setfacl -m u:'toby':rx '/mnt/shared/s/Secret'")
	iDef := strings.Index(cmd, "setfacl -m d:u:'toby':rx '/mnt/shared/s/Secret'")
	if iBlank < 0 || iChmod < 0 || iAcc < 0 || iDef < 0 {
		t.Fatalf("missing expected fragments in: %q", cmd)
	}
	if !(iBlank < iChmod && iChmod < iAcc) {
		t.Fatalf("fragments out of order (want blank<chmod<access): %q", cmd)
	}
	if !strings.Contains(cmd, "setfacl -m u:'toby':x '/mnt/shared/s'") {
		t.Fatalf("missing share-root traverse grant: %q", cmd)
	}
}

func TestBuildLockCmd_RecursiveNeverRecursesDefaultAcl(t *testing.T) {
	cmd := buildLockCmd("-R ", "/mnt/shared/s", "/mnt/shared/s/Secret", "rwx", []string{"toby"})
	if !strings.Contains(cmd, "setfacl -R -b '/mnt/shared/s/Secret'") {
		t.Fatalf("expected recursive wipe: %q", cmd)
	}
	if !strings.Contains(cmd, "chmod -R o= '/mnt/shared/s/Secret'") {
		t.Fatalf("expected recursive chmod: %q", cmd)
	}
	// access ACL recurses; default ACL must NOT (setfacl errors on files with -R + d:)
	if !strings.Contains(cmd, "setfacl -R -m u:'toby':rwx '/mnt/shared/s/Secret'") {
		t.Fatalf("expected recursive access grant: %q", cmd)
	}
	if strings.Contains(cmd, "-R -m d:") {
		t.Fatalf("default ACL must never carry -R: %q", cmd)
	}
	if !strings.Contains(cmd, "setfacl -m d:u:'toby':rwx '/mnt/shared/s/Secret'") {
		t.Fatalf("expected non-recursive default grant: %q", cmd)
	}
}

func TestBuildUnlockCmd_ReopensAndRegrantsValidUsers(t *testing.T) {
	cmd := buildUnlockCmd("", "/mnt/shared/s/Secret", []string{"alice", "@devs", "IT\\bob"})
	for _, want := range []string{
		"setfacl -b '/mnt/shared/s/Secret'",
		"chmod o+rX '/mnt/shared/s/Secret'",
		"setfacl -m u:'alice':rwX '/mnt/shared/s/Secret'",
		"setfacl -m d:u:'alice':rwX '/mnt/shared/s/Secret'",
		"setfacl -m g:'devs':rwX '/mnt/shared/s/Secret'",
		"setfacl -m d:g:'devs':rwX '/mnt/shared/s/Secret'",
		"setfacl -m u:'IT\\bob':rwX '/mnt/shared/s/Secret'",
		"setfacl -m d:u:'IT\\bob':rwX '/mnt/shared/s/Secret'",
	} {
		if !strings.Contains(cmd, want) {
			t.Fatalf("missing %q in: %q", want, cmd)
		}
	}
}

func TestAclSpec(t *testing.T) {
	if got := aclSpec("@devs"); got != "g:'devs'" {
		t.Fatalf("group spec = %q", got)
	}
	if got := aclSpec("IT\\bob"); got != "u:'IT\\bob'" {
		t.Fatalf("user spec = %q", got)
	}
}

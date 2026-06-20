package ldap_test

import (
	"encoding/json"
	"strings"
	"testing"

	"smb-server/backend/internal/ldap"
	"smb-server/backend/internal/samba"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// captureExec wraps FakeExecutor and returns a fixed output for every Execute call.
// It still records calls in FakeExecutor.Calls so tests can inspect them.
type captureExec struct {
	*samba.FakeExecutor
	output string
}

func (e *captureExec) Execute(cmd string) samba.ExecResult {
	e.FakeExecutor.Execute(cmd) // records in Calls
	return samba.ExecResult{Success: true, Output: e.output}
}

func newCE(output string) *captureExec {
	return &captureExec{FakeExecutor: samba.NewFakeExecutor(), output: output}
}

func newLDAPSvc(exec samba.Executor) *ldap.Service {
	return ldap.NewService(exec, ldap.Config{
		Server: "10.0.0.1",
		Port:   389,
		BaseDN: "DC=test,DC=com",
		BindDN: "admin@test.com",
		BindPW: "s3cr3t!p@ss'word", // contains single-quote to test escaping
		Domain: "TEST.COM",
	})
}

// ── TestConnection ──────────────────────────────────────────────────────────

func TestTestConnection_Connected(t *testing.T) {
	ce := newCE("dn: DC=test,DC=com\n")
	status := newLDAPSvc(ce).TestConnection()
	assert.True(t, status.Connected)
	assert.Empty(t, status.Error)
	assert.Equal(t, "TEST.COM", status.Domain)
}

func TestTestConnection_InvalidCredentials(t *testing.T) {
	ce := newCE("ldap_bind: Invalid credentials (49)")
	status := newLDAPSvc(ce).TestConnection()
	assert.False(t, status.Connected)
	assert.Equal(t, "Invalid credentials", status.Error)
}

func TestTestConnection_CannotConnect(t *testing.T) {
	ce := newCE("ldap_sasl_interactive_bind_s: Can't contact LDAP server (-1)")
	status := newLDAPSvc(ce).TestConnection()
	assert.False(t, status.Connected)
	assert.Equal(t, "Cannot connect to LDAP server", status.Error)
}

func TestTestConnection_ErrorOutputTruncatedTo100(t *testing.T) {
	ce := newCE(strings.Repeat("X", 150))
	status := newLDAPSvc(ce).TestConnection()
	assert.False(t, status.Connected)
	assert.LessOrEqual(t, len(status.Error), 110, "error output must be truncated")
	assert.Contains(t, status.Error, "…")
}

// ── ConnectionStatus JSON shape ─────────────────────────────────────────────

// TestConnectionStatus_HidesSensitiveFields ensures the JSON response from
// /api/ad/status never exposes server IP, BindDN, or BaseDN.
func TestConnectionStatus_HidesSensitiveFields(t *testing.T) {
	ce := newCE("dn: DC=test,DC=com\n")
	status := newLDAPSvc(ce).TestConnection()

	data, err := json.Marshal(status)
	require.NoError(t, err)
	body := string(data)

	assert.NotContains(t, body, "ldap_server", "server IP must not appear in JSON")
	assert.NotContains(t, body, "bind_dn", "BindDN must not appear in JSON")
	assert.NotContains(t, body, "base_dn", "BaseDN must not appear in JSON")
	assert.Contains(t, body, `"domain"`, "domain field must be present")
	assert.Contains(t, body, "TEST.COM")
}

// ── Command security ─────────────────────────────────────────────────────────

// TestLDAPSearch_PasswordNotInCommand verifies the bind password is never
// passed as a -w argument to ldapsearch (which would be visible in `ps aux`).
// The fix writes the password to a temp file via `printf` (a shell builtin —
// no separate process) and uses ldapsearch -y <file>.
func TestLDAPSearch_PasswordNotInCommand(t *testing.T) {
	ce := newCE("dn: DC=test,DC=com\n")
	newLDAPSvc(ce).TestConnection()

	require.Len(t, ce.Calls, 1, "expected one Execute call")
	cmd := ce.Calls[0]

	// The whole-command checks
	assert.Contains(t, cmd, "-y", "must use -y (file) not -w (argument)")
	assert.NotContains(t, cmd, "-w ", "raw -w flag must not appear anywhere")

	// The ldapsearch process itself must not receive the password.
	// `printf` (shell builtin) is allowed to have the escaped password in the
	// command string; what matters is that `ldapsearch` args do not.
	ldapsearchIdx := strings.Index(cmd, "ldapsearch")
	require.True(t, ldapsearchIdx >= 0, "command must contain ldapsearch")
	ldapsearchPart := cmd[ldapsearchIdx:]

	assert.NotContains(t, ldapsearchPart, "s3cr3t", "password must not be in ldapsearch arguments")
	assert.Contains(t, ldapsearchPart, "-y ", "ldapsearch must use -y for password file")
}

// TestLDAPSearch_SingleQuoteInPasswordEscaped verifies that a password
// containing a single quote doesn't break the shell command.
func TestLDAPSearch_SingleQuoteInPasswordEscaped(t *testing.T) {
	ce := newCE("dn: DC=test,DC=com\n")
	newLDAPSvc(ce).TestConnection()
	// If single quote escaping is broken, bash would fail to parse the command
	// and FakeExecutor would still succeed (it's a fake), but the real test
	// is that the raw unescaped password string doesn't appear in the command.
	require.Len(t, ce.Calls, 1)
	cmd := ce.Calls[0]
	// The raw password with unescaped single-quote should not appear verbatim
	assert.NotContains(t, cmd, "s3cr3t!p@ss'word")
}

// ── SearchUsers ──────────────────────────────────────────────────────────────

func TestSearchUsers_ParsesADUsers(t *testing.T) {
	ldifOutput := `
dn: CN=Alice Smith,OU=Staff,DC=test,DC=com
sAMAccountName: alice
cn: Alice Smith
mail: alice@test.com

dn: CN=Bob Jones,OU=Lecturer,DC=test,DC=com
sAMAccountName: bob
cn: Bob Jones

`
	ce := newCE(ldifOutput)
	users, err := newLDAPSvc(ce).SearchUsers("", "", 0)
	require.NoError(t, err)
	require.Len(t, users, 2)

	assert.Equal(t, `TEST\alice`, users[0].Username)
	assert.Equal(t, "Alice Smith", users[0].DisplayName)
	assert.Equal(t, "alice@test.com", users[0].Email)
	assert.Equal(t, "Staff", users[0].OU)

	assert.Equal(t, `TEST\bob`, users[1].Username)
	assert.Equal(t, "Lecturer", users[1].OU)
}

func TestSearchUsers_SkipsMachineAccounts(t *testing.T) {
	ldifOutput := `
dn: CN=PC01,DC=test,DC=com
sAMAccountName: PC01$

dn: CN=Alice,DC=test,DC=com
sAMAccountName: alice

`
	ce := newCE(ldifOutput)
	users, err := newLDAPSvc(ce).SearchUsers("", "", 0)
	require.NoError(t, err)
	require.Len(t, users, 1)
	assert.Equal(t, `TEST\alice`, users[0].Username)
}

func TestSearchUsers_RespectsLimit(t *testing.T) {
	ldifOutput := `
dn: CN=Alice,DC=test,DC=com
sAMAccountName: alice

dn: CN=Bob,DC=test,DC=com
sAMAccountName: bob

dn: CN=Carol,DC=test,DC=com
sAMAccountName: carol

`
	ce := newCE(ldifOutput)
	users, err := newLDAPSvc(ce).SearchUsers("", "", 2)
	require.NoError(t, err)
	assert.Len(t, users, 2)
}

// ── SearchGroups ─────────────────────────────────────────────────────────────

func TestSearchGroups_ParsesGroups(t *testing.T) {
	ldifOutput := `
dn: CN=IT Staff,OU=Group,DC=test,DC=com
cn: IT Staff
description: Information Technology Staff

dn: CN=Students,OU=Group,DC=test,DC=com
cn: Students

`
	ce := newCE(ldifOutput)
	groups, err := newLDAPSvc(ce).SearchGroups("", 0)
	require.NoError(t, err)
	require.Len(t, groups, 2)

	assert.Equal(t, "IT Staff", groups[0].Name)
	assert.Equal(t, "@IT Staff", groups[0].SMBName)
	assert.Equal(t, "Information Technology Staff", groups[0].Description)

	assert.Equal(t, "Students", groups[1].Name)
	assert.Equal(t, "@Students", groups[1].SMBName)
}

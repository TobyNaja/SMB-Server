package samba_test

import (
	"strings"
	"testing"

	"smb-server/backend/internal/samba"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Compile-time: FakeExecutor must satisfy the full Executor interface.
var _ samba.Executor = (*samba.FakeExecutor)(nil)

func TestCreateUser_PasswordNotInCallLog(t *testing.T) {
	f := samba.NewFakeExecutor()
	f.CreateUser("alice", "topsecret!p@ss")

	for _, call := range f.Calls {
		assert.NotContains(t, call, "topsecret", "password must not appear in any recorded call")
	}
	require.Len(t, f.Users, 1)
	assert.Equal(t, "alice", f.Users[0].Username)
}

func TestSetPassword_PasswordNotInCallLog(t *testing.T) {
	f := samba.NewFakeExecutor()
	f.SetPassword("alice", "n3wPassw0rd!")

	for _, call := range f.Calls {
		assert.NotContains(t, call, "n3wPassw0rd", "password must not appear in any recorded call")
	}
}

func TestExecuteWithInput_RecordedWithoutInput(t *testing.T) {
	f := samba.NewFakeExecutor()
	result := f.ExecuteWithInput([]string{"smbpasswd", "-s", "alice"}, "secret\nsecret\n")

	assert.True(t, result.Success)
	require.Len(t, f.Calls, 1)
	// Command is recorded but the input (password) is not
	assert.Contains(t, f.Calls[0], "smbpasswd")
	assert.NotContains(t, f.Calls[0], "secret")
	// Verify it's an ExecuteWithInput call, not a plain shell Execute call
	assert.True(t, strings.HasPrefix(f.Calls[0], "ExecuteWithInput:"))
}

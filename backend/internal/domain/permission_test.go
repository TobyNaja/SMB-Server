package domain_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"smb-server/backend/internal/domain"
)

func TestSyncPermissions_InvalidUsersRemovedFromAll(t *testing.T) {
	in := domain.SharePerms{
		Valid:   []string{"alice", "bob"},
		Write:   []string{"alice"},
		Admin:   []string{"bob"},
		Invalid: []string{"alice"},
	}
	out := domain.SyncPermissions(in)
	assert.NotContains(t, out.Valid, "alice")
	assert.NotContains(t, out.Write, "alice")
	assert.NotContains(t, out.Admin, "alice")
	assert.Contains(t, out.Invalid, "alice")
}

func TestSyncPermissions_AdminAutoAddedToValid(t *testing.T) {
	in := domain.SharePerms{Admin: []string{"carol"}}
	out := domain.SyncPermissions(in)
	assert.Contains(t, out.Valid, "carol")
	assert.NotContains(t, out.Write, "carol")
}

func TestSyncPermissions_WriteAutoAddedToValid(t *testing.T) {
	in := domain.SharePerms{Write: []string{"dave"}}
	out := domain.SyncPermissions(in)
	assert.Contains(t, out.Valid, "dave")
	assert.NotContains(t, out.Read, "dave")
}

func TestSyncPermissions_ReadAutoAddedToValid(t *testing.T) {
	in := domain.SharePerms{Read: []string{"eve"}}
	out := domain.SyncPermissions(in)
	assert.Contains(t, out.Valid, "eve")
}

func TestSyncPermissions_PureFunction_DoesNotMutateInput(t *testing.T) {
	in := domain.SharePerms{Write: []string{"frank"}, Read: []string{"frank"}}
	_ = domain.SyncPermissions(in)
	assert.Contains(t, in.Read, "frank") // input unchanged
}

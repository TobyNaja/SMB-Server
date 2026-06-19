package samba_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"smb-server/backend/internal/samba"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newParser(t *testing.T) (*samba.SmbConfParser, string) {
	t.Helper()
	dir := t.TempDir()
	sharesPath := filepath.Join(dir, "shares.conf")
	globalPath := filepath.Join(dir, "smb.conf")
	// Write a minimal smb.conf
	_ = os.WriteFile(globalPath, []byte("[global]\n    workgroup = IT\n    realm = IT.KMITL.AC.TH\n"), 0o644)
	return samba.NewSmbConfParser(sharesPath, globalPath), sharesPath
}

func TestCreateAndGetShare(t *testing.T) {
	p, _ := newParser(t)
	ok := p.CreateShare("testshare", "/srv/testshare", "Test share")
	require.True(t, ok)

	share, err := p.GetShare("testshare")
	require.NoError(t, err)
	require.NotNil(t, share)
	assert.Equal(t, "testshare", share.Name)
	assert.Equal(t, "/srv/testshare", share.Path)
	assert.Equal(t, "Test share", share.Comment)
}

func TestCreateShare_DuplicateRejected(t *testing.T) {
	p, _ := newParser(t)
	p.CreateShare("dup", "/srv/dup", "")
	ok := p.CreateShare("dup", "/srv/dup2", "")
	assert.False(t, ok)
}

func TestDeleteShare(t *testing.T) {
	p, _ := newParser(t)
	p.CreateShare("todelete", "/srv/todelete", "")
	ok := p.DeleteShare("todelete")
	require.True(t, ok)
	share, _ := p.GetShare("todelete")
	assert.Nil(t, share)
}

func TestRoundTrip_ParseAndSaveAndReload(t *testing.T) {
	p, sharesPath := newParser(t)
	p.CreateShare("myshare", "/srv/myshare", "My share")
	p.SetWriteList("myshare", []string{"alice", `IT\bob`})

	// Reload from disk
	p2 := samba.NewSmbConfParser(sharesPath, filepath.Dir(sharesPath)+"/smb.conf")
	share, err := p2.GetShare("myshare")
	require.NoError(t, err)
	require.NotNil(t, share)
	assert.Contains(t, share.WriteList, "alice")
	assert.Contains(t, share.WriteList, `IT\bob`)
}

func TestHeaderCommentWritten(t *testing.T) {
	p, sharesPath := newParser(t)
	p.CreateShare("s", "/srv/s", "")
	data, err := os.ReadFile(sharesPath)
	require.NoError(t, err)
	assert.True(t, strings.Contains(string(data), "SMB Shares - Managed by SMB Manager Web UI"))
}

func TestSetPermissionsTriggersMatrix(t *testing.T) {
	p, _ := newParser(t)
	p.CreateShare("shared", "/srv/shared", "")

	// Put alice in both write and read; matrix should remove her from read
	p.SetWriteList("shared", []string{"alice"})
	p.SetReadList("shared", []string{"alice", "bob"})

	share, _ := p.GetShare("shared")
	assert.Contains(t, share.WriteList, "alice")
	assert.NotContains(t, share.ReadList, "alice") // matrix removed duplicate
	assert.Contains(t, share.ReadList, "bob")
	assert.Contains(t, share.ValidUsers, "alice")
	assert.Contains(t, share.ValidUsers, "bob")
}

func TestSetInvalidUsers_EvictsFromAll(t *testing.T) {
	p, _ := newParser(t)
	p.CreateShare("secure", "/srv/secure", "")
	p.SetWriteList("secure", []string{"eve", "alice"})
	p.SetInvalidUsers("secure", []string{"eve"})

	share, _ := p.GetShare("secure")
	assert.NotContains(t, share.WriteList, "eve")
	assert.NotContains(t, share.ValidUsers, "eve")
	assert.Contains(t, share.InvalidUsers, "eve")
	assert.Contains(t, share.WriteList, "alice")
}

func TestGetGlobal(t *testing.T) {
	p, _ := newParser(t)
	g := p.GetGlobal()
	assert.Equal(t, "IT", g["workgroup"])
}

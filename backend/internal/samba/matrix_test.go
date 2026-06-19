package samba_test

import (
	"testing"

	"smb-server/backend/internal/samba"

	"github.com/stretchr/testify/assert"
)

func TestSyncPermissions(t *testing.T) {
	tests := []struct {
		name  string
		input samba.SharePerms
		want  samba.SharePerms
	}{
		{
			name:  "empty input",
			input: samba.SharePerms{},
			want:  samba.SharePerms{},
		},
		{
			name: "invalid_users evicted from all lists",
			input: samba.SharePerms{
				Valid:   []string{"alice", "bob"},
				Write:   []string{"alice"},
				Read:    []string{"bob"},
				Admin:   []string{"carol"},
				Invalid: []string{"alice", "bob", "carol"},
			},
			want: samba.SharePerms{
				Valid:   []string{},
				Write:   []string{},
				Read:    []string{},
				Admin:   []string{},
				Invalid: []string{"alice", "bob", "carol"},
			},
		},
		{
			name: "admin_users removed from write and read, added to valid",
			input: samba.SharePerms{
				Write: []string{"alice"},
				Read:  []string{"alice"},
				Admin: []string{"alice"},
			},
			want: samba.SharePerms{
				Valid:   []string{"alice"},
				Write:   []string{},
				Read:    []string{},
				Admin:   []string{"alice"},
				Invalid: []string{},
			},
		},
		{
			name: "write_list removed from read_list, added to valid",
			input: samba.SharePerms{
				Write: []string{"alice"},
				Read:  []string{"alice", "bob"},
			},
			want: samba.SharePerms{
				Valid:   []string{"alice", "bob"},
				Write:   []string{"alice"},
				Read:    []string{"bob"},
				Admin:   []string{},
				Invalid: []string{},
			},
		},
		{
			name: "read_list added to valid",
			input: samba.SharePerms{
				Read: []string{"bob"},
			},
			want: samba.SharePerms{
				Valid:   []string{"bob"},
				Write:   []string{},
				Read:    []string{"bob"},
				Admin:   []string{},
				Invalid: []string{},
			},
		},
		{
			name: "invalid has highest priority — overrides admin",
			input: samba.SharePerms{
				Admin:   []string{"eve"},
				Invalid: []string{"eve"},
			},
			want: samba.SharePerms{
				Valid:   []string{},
				Admin:   []string{},
				Invalid: []string{"eve"},
			},
		},
		{
			name: "write user auto-added to valid even if valid was empty",
			input: samba.SharePerms{
				Write: []string{"alice"},
			},
			want: samba.SharePerms{
				Valid: []string{"alice"},
				Write: []string{"alice"},
			},
		},
		{
			name: "AD domain users and @groups preserved",
			input: samba.SharePerms{
				Write: []string{`IT\alice`, "@Domain Users"},
			},
			want: samba.SharePerms{
				Valid: []string{`IT\alice`, "@Domain Users"},
				Write: []string{`IT\alice`, "@Domain Users"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := samba.SyncPermissions(tt.input)
			// Nil and empty slice are equivalent for these assertions
			if len(tt.want.Valid) == 0 {
				assert.Empty(t, got.Valid)
			} else {
				assert.Equal(t, tt.want.Valid, got.Valid)
			}
			if len(tt.want.Write) == 0 {
				assert.Empty(t, got.Write)
			} else {
				assert.Equal(t, tt.want.Write, got.Write)
			}
			if len(tt.want.Read) == 0 {
				assert.Empty(t, got.Read)
			} else {
				assert.Equal(t, tt.want.Read, got.Read)
			}
			if len(tt.want.Admin) == 0 {
				assert.Empty(t, got.Admin)
			} else {
				assert.Equal(t, tt.want.Admin, got.Admin)
			}
			if len(tt.want.Invalid) == 0 {
				assert.Empty(t, got.Invalid)
			} else {
				assert.Equal(t, tt.want.Invalid, got.Invalid)
			}
		})
	}
}

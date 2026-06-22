// backend/internal/adapter/fake/fake_test.go
package fake_test

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "smb-server/backend/internal/adapter/fake"
    "smb-server/backend/internal/domain"
    "smb-server/backend/internal/port"
)

func TestFakeCommandRunner_ImplementsInterface(t *testing.T) {
    var _ port.CommandRunner = fake.NewCommandRunner()
}

func TestFakeCommandRunner_RecordsCalls(t *testing.T) {
    f := fake.NewCommandRunner()
    f.Execute("chmod 0750 /data/test")
    assert.Contains(t, f.Calls, "chmod 0750 /data/test")
}

func TestInMemoryShareStore_ImplementsInterface(t *testing.T) {
    var _ port.ShareStore = fake.NewShareStore()
}

func TestInMemoryShareStore_CreateAndGet(t *testing.T) {
    s := fake.NewShareStore()
    _ = s.CreateShare("myshare", "/data/myshare", "test share")
    share, err := s.GetShare("myshare")
    assert.NoError(t, err)
    assert.Equal(t, "myshare", share.Name)
    assert.Equal(t, "/data/myshare", share.Path)

    // duplicate create returns ErrAlreadyExists
    err = s.CreateShare("myshare", "/data/myshare", "test share")
    assert.Equal(t, domain.ErrAlreadyExists, err)

    // missing share returns ErrNotFound
    _, err2 := s.GetShare("nonexistent")
    assert.Equal(t, domain.ErrNotFound, err2)
}

func TestInMemoryAuditLog_ImplementsInterface(t *testing.T) {
    var _ port.AuditLog = fake.NewAuditLog()
}

func TestInMemoryAuditLog_AppendAndQuery(t *testing.T) {
    l := fake.NewAuditLog()
    l.Append(domain.NewAuditEntry("create_share", "admin", "share", "test", "success", "127.0.0.1"))
    entries, err := l.Query(10, "", "")
    assert.NoError(t, err)
    assert.Len(t, entries, 1)
    assert.Equal(t, "create_share", entries[0].Action)

    // filter by action — wrong action returns empty
    filtered, _ := l.Query(10, "delete_share", "")
    assert.Len(t, filtered, 0)

    // limit truncates oldest entries
    l.Append(domain.NewAuditEntry("create_share", "admin", "share", "test2", "success", "127.0.0.1"))
    limited, _ := l.Query(1, "", "")
    assert.Len(t, limited, 1)
}

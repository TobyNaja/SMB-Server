package ldap_test

import (
	"encoding/base64"
	"fmt"
	"testing"

	"smb-server/backend/internal/ldap"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseLDIF_Basic(t *testing.T) {
	input := `
dn: CN=Alice,OU=Staff,DC=it,DC=kmitl,DC=ac,DC=th
sAMAccountName: alice
cn: Alice Smith
mail: alice@it.kmitl.ac.th

`
	entries := ldap.ParseLDIF(input)
	require.Len(t, entries, 1)
	assert.Equal(t, "alice", ldap.GetString(entries[0], "samaccountname"))
	assert.Equal(t, "Alice Smith", ldap.GetString(entries[0], "cn"))
	assert.Equal(t, "alice@it.kmitl.ac.th", ldap.GetString(entries[0], "mail"))
}

func TestParseLDIF_Base64Value(t *testing.T) {
	name := "สมศักดิ์ ใจดี"
	encoded := base64.StdEncoding.EncodeToString([]byte(name))
	input := fmt.Sprintf("dn: CN=test,DC=it,DC=kmitl,DC=ac,DC=th\ncn:: %s\n\n", encoded)

	entries := ldap.ParseLDIF(input)
	require.Len(t, entries, 1)
	assert.Equal(t, name, ldap.GetString(entries[0], "cn"))
}

func TestParseLDIF_RepeatedAttribute(t *testing.T) {
	input := `
dn: CN=Bob,DC=it,DC=kmitl,DC=ac,DC=th
memberOf: CN=GroupA,DC=it,DC=kmitl,DC=ac,DC=th
memberOf: CN=GroupB,DC=it,DC=kmitl,DC=ac,DC=th

`
	entries := ldap.ParseLDIF(input)
	require.Len(t, entries, 1)
	groups := ldap.GetStrings(entries[0], "memberof")
	require.Len(t, groups, 2)
	assert.Contains(t, groups, "CN=GroupA,DC=it,DC=kmitl,DC=ac,DC=th")
	assert.Contains(t, groups, "CN=GroupB,DC=it,DC=kmitl,DC=ac,DC=th")
}

func TestParseLDIF_MultiLineContinuation(t *testing.T) {
	// ldapsearch wraps long lines with a leading space
	input := "dn: CN=LongName,OU=Very Long Organisational Unit,DC=it,DC=kmitl,DC=ac,DC=th\n" +
		" ,DC=extra\n" +
		"cn: LongName\n\n"
	entries := ldap.ParseLDIF(input)
	require.Len(t, entries, 1)
	assert.Contains(t, ldap.GetString(entries[0], "dn"), "DC=extra")
}

func TestParseLDIF_SkipsEntriesWithoutDN(t *testing.T) {
	input := "cn: orphan\nmail: orphan@example.com\n\n"
	entries := ldap.ParseLDIF(input)
	assert.Len(t, entries, 0)
}

func TestParseLDIF_MultipleEntries(t *testing.T) {
	input := `
dn: CN=Alice,DC=it,DC=kmitl,DC=ac,DC=th
sAMAccountName: alice

dn: CN=Bob,DC=it,DC=kmitl,DC=ac,DC=th
sAMAccountName: bob

`
	entries := ldap.ParseLDIF(input)
	assert.Len(t, entries, 2)
}

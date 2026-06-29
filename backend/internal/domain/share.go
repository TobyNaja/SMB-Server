package domain

// Share is the canonical representation of a Samba share.
type Share struct {
	Name          string
	Path          string
	Comment       string
	Browseable    bool
	ReadOnly      bool
	GuestOK       bool
	ABSE          bool
	ValidUsers    []string
	WriteList     []string
	ReadList      []string
	AdminUsers    []string
	InvalidUsers  []string
	CreateMask    string
	DirectoryMask string
}

// CreateShareRequest carries validated input for share creation.
type CreateShareRequest struct {
	Name       string
	Path       string
	Comment    string
	Browseable bool
	GuestOK    bool
	ABSE       bool
	Actor      string
}

// SharePerms holds the five user lists subject to the permission matrix.
type SharePerms struct {
	Valid   []string
	Write   []string
	Read    []string
	Admin   []string
	Invalid []string
}

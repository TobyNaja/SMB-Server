package samba

// SharePerms holds the five user lists for a share.
type SharePerms struct {
	Valid   []string
	Write   []string
	Read    []string
	Admin   []string
	Invalid []string
}

// union returns the union of two slices (preserving order of a, then unique b).
func union(a, b []string) []string {
	seen := make(map[string]bool, len(a))
	result := make([]string, 0, len(a)+len(b))
	for _, v := range a {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	for _, v := range b {
		if !seen[v] {
			seen[v] = true
			result = append(result, v)
		}
	}
	return result
}

// exclude returns a with every element of b removed.
func exclude(a, b []string) []string {
	rm := make(map[string]bool, len(b))
	for _, v := range b {
		rm[v] = true
	}
	result := make([]string, 0, len(a))
	for _, v := range a {
		if !rm[v] {
			result = append(result, v)
		}
	}
	return result
}

// SyncPermissions enforces the permission matrix rules:
//  1. invalid_users — removed from all other lists (highest priority).
//  2. admin_users — removed from write_list and read_list; unioned into valid_users.
//  3. write_list — removed from read_list; unioned into valid_users.
//  4. read_list — unioned into valid_users.
//
// All lists are sanitized before being returned.
func SyncPermissions(p SharePerms) SharePerms {
	valid := p.Valid
	write := p.Write
	read := p.Read
	admin := p.Admin
	invalid := p.Invalid

	// 1. invalid_users — evict from everything
	if len(invalid) > 0 {
		valid = exclude(valid, invalid)
		write = exclude(write, invalid)
		read = exclude(read, invalid)
		admin = exclude(admin, invalid)
	}

	// 2. admin_users — remove from write/read; auto-add to valid
	if len(admin) > 0 {
		write = exclude(write, admin)
		read = exclude(read, admin)
		valid = union(valid, admin)
	}

	// 3. write_list — remove from read; auto-add to valid
	if len(write) > 0 {
		read = exclude(read, write)
		valid = union(valid, write)
	}

	// 4. read_list — auto-add to valid
	if len(read) > 0 {
		valid = union(valid, read)
	}

	return SharePerms{
		Valid:   sanitizeUsers(valid),
		Write:   sanitizeUsers(write),
		Read:    sanitizeUsers(read),
		Admin:   sanitizeUsers(admin),
		Invalid: sanitizeUsers(invalid),
	}
}

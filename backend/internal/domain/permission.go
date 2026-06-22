package domain

// SyncPermissions enforces permission priority rules (pure function — no I/O).
//  1. invalid_users — removed from all other lists.
//  2. admin_users   — removed from write/read; unioned into valid.
//  3. write_list    — removed from read; unioned into valid.
//  4. read_list     — unioned into valid.
func SyncPermissions(p SharePerms) SharePerms {
	valid, write, read, admin, invalid :=
		p.Valid, p.Write, p.Read, p.Admin, p.Invalid

	if len(invalid) > 0 {
		valid = exclude(valid, invalid)
		write = exclude(write, invalid)
		read = exclude(read, invalid)
		admin = exclude(admin, invalid)
	}
	if len(admin) > 0 {
		write = exclude(write, admin)
		read = exclude(read, admin)
		valid = union(valid, admin)
	}
	if len(write) > 0 {
		read = exclude(read, write)
		valid = union(valid, write)
	}
	if len(read) > 0 {
		valid = union(valid, read)
	}
	return SharePerms{Valid: valid, Write: write, Read: read, Admin: admin, Invalid: invalid}
}

func union(a, b []string) []string {
	seen := make(map[string]bool, len(a))
	out := make([]string, 0, len(a)+len(b))
	for _, v := range a {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	for _, v := range b {
		if !seen[v] {
			seen[v] = true
			out = append(out, v)
		}
	}
	return out
}

func exclude(a, b []string) []string {
	rm := make(map[string]bool, len(b))
	for _, v := range b {
		rm[v] = true
	}
	out := make([]string, 0, len(a))
	for _, v := range a {
		if !rm[v] {
			out = append(out, v)
		}
	}
	return out
}

package httpapi

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/gofiber/fiber/v2"
)

var (
	// perms: ordered subset of rwx (e.g. "r", "rx", "rwx")
	subfolderValidPerms = regexp.MustCompile(`^r?w?x?$`)
	// username: block shell metachars / quotes; allow DOMAIN\user, dots, @, space
	subfolderValidUser = regexp.MustCompile(`^[A-Za-z0-9_.@\\ -]+$`)
	// subfolder path: allowlist only — folder names, slashes, spaces, dots, dashes.
	// Blocks quotes/$/;/backtick etc. so the value is safe to single-quote in a
	// shell command. Empty = share root. Traversal is caught separately by Rel.
	subfolderValidPath = regexp.MustCompile(`^[A-Za-z0-9 ._/-]*$`)
)

type subfolderPermissionRequest struct {
	SubfolderPath string `json:"subfolder_path"`
	Username      string `json:"username"`
	Permissions   string `json:"permissions"` // "rwx","rx","r","" or "none" = remove
	Recursive     bool   `json:"recursive"`
}

// subfolderLockRequest makes a subfolder private to exactly the listed users.
type subfolderLockRequest struct {
	SubfolderPath string   `json:"subfolder_path"`
	Users         []string `json:"users"`       // exact allowlist (empty = owner only)
	Permissions   string   `json:"permissions"` // perms per allowed user; default "rx"
	Recursive     bool     `json:"recursive"`
}

type subfolderUnlockRequest struct {
	SubfolderPath string `json:"subfolder_path"`
	Recursive     bool   `json:"recursive"`
}

// resolveSubfolder validates the path stays inside the share root.
func (h *sharesHandlers) resolveSubfolder(name, sub string) (base, target, rel string, err error) {
	p := h.parser()
	share, err := p.GetShare(name)
	if err != nil || share == nil {
		return "", "", "", fmt.Errorf("share not found")
	}
	if !subfolderValidPath.MatchString(sub) {
		return "", "", "", fmt.Errorf("invalid subfolder path")
	}
	base = share.Path
	target = filepath.Join(base, filepath.Clean("/"+sub))
	rel, err = filepath.Rel(base, target)
	if err != nil || rel == ".." || strings.HasPrefix(rel, "../") {
		return "", "", "", fmt.Errorf("invalid subfolder path")
	}
	return base, target, rel, nil
}

func (h *sharesHandlers) updateSubfolderPermissions(c *fiber.Ctx) error {
	name := c.Params("name")
	var req subfolderPermissionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid request body"})
	}

	username := strings.TrimSpace(req.Username)
	if username == "" || !subfolderValidUser.MatchString(username) {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid username"})
	}
	perms := strings.TrimSpace(req.Permissions)
	if perms != "" && perms != "none" && !subfolderValidPerms.MatchString(perms) {
		return c.Status(400).JSON(fiber.Map{"detail": "permissions must be an ordered subset of rwx"})
	}

	basePath, targetPath, rel, err := h.resolveSubfolder(name, req.SubfolderPath)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": err.Error()})
	}

	qUser := "'" + username + "'"
	qPath := "'" + targetPath + "'"
	recursiveFlag := ""
	if req.Recursive {
		recursiveFlag = "-R "
	}

	var cmd string
	if perms == "" || perms == "none" {
		// remove both access ACL and default ACL (no ghost perms on new files)
		cmd = fmt.Sprintf(
			"setfacl %s-x u:%s %s 2>/dev/null; setfacl %s-x d:u:%s %s 2>/dev/null; true",
			recursiveFlag, qUser, qPath,
			recursiveFlag, qUser, qPath,
		)
	} else {
		// Access ACL recurses if asked; the default ACL (inheritance) is valid on
		// directories only, so it must never carry -R — setfacl errors on regular
		// files when handed a default entry, which would abort the whole && chain.
		cmd = fmt.Sprintf("setfacl %s-m u:%s:%s %s", recursiveFlag, qUser, perms, qPath)
		cmd += fmt.Sprintf(" && setfacl -m d:u:%s:%s %s", qUser, perms, qPath)
		// grant traverse (x) up the parent chain so the user can reach it
		dir := filepath.Dir(targetPath)
		for strings.HasPrefix(dir, basePath) && dir != basePath {
			cmd += fmt.Sprintf(" && setfacl -m u:%s:x '%s'", qUser, dir)
			dir = filepath.Dir(dir)
		}
		cmd += fmt.Sprintf(" && setfacl -m u:%s:x '%s'", qUser, basePath)
	}

	if out, err := h.exec.ExecuteOutput(cmd); err != nil {
		return c.Status(500).JSON(fiber.Map{"detail": "setfacl failed: " + out})
	}
	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("permissions for %q on %q updated", username, rel),
	})
}

func (h *sharesHandlers) getSubfolderPermissions(c *fiber.Ctx) error {
	name := c.Params("name")
	sub := c.Query("path", ".")

	_, targetPath, rel, err := h.resolveSubfolder(name, sub)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": err.Error()})
	}

	out, err := h.exec.ExecuteOutput("getfacl -p '" + targetPath + "'")
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"detail": "getfacl failed: " + out})
	}

	type aclEntry struct {
		Type    string `json:"type"`    // user | group
		Name    string `json:"name"`    // empty = owner/owning-group
		Perms   string `json:"perms"`   // e.g. rwx, r-x
		Default bool   `json:"default"` // inheritance entry
	}
	var entries []aclEntry
	otherPerms := "" // access "other::" class — drives the locked flag
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		isDefault := strings.HasPrefix(line, "default:")
		line = strings.TrimPrefix(line, "default:")
		// The access other:: class tells us whether unauthorized users are shut out.
		if !isDefault && strings.HasPrefix(line, "other::") {
			otherPerms = strings.SplitN(line, "::", 2)[1]
			if i := strings.IndexAny(otherPerms, " \t#"); i >= 0 {
				otherPerms = otherPerms[:i]
			}
		}
		parts := strings.Split(line, ":")
		if len(parts) < 3 {
			continue
		}
		kind := parts[0]
		if kind != "user" && kind != "group" {
			continue
		}
		// name may itself contain ':' (DOMAIN\user won't, but be safe):
		nameField := strings.Join(parts[1:len(parts)-1], ":")
		permField := parts[len(parts)-1]
		// strip getfacl effective-rights comments, e.g. "rwx\t#effective:r-x"
		if i := strings.IndexAny(permField, " \t#"); i >= 0 {
			permField = permField[:i]
		}
		entries = append(entries, aclEntry{
			Type: kind, Name: nameField, Perms: permField, Default: isDefault,
		})
	}
	// Locked = unauthorized users have no traverse/read via the other:: class.
	locked := !strings.ContainsAny(otherPerms, "rx")
	return c.JSON(fiber.Map{"share": name, "path": rel, "entries": entries, "locked": locked})
}

// aclSpec turns a Samba user/group token into a setfacl principal spec.
// "@grp" -> g:'grp'  ;  "IT\user" -> u:'IT\user'. The name is single-quoted;
// callers must pre-validate it with subfolderValidUser (no quote chars pass).
func aclSpec(u string) string {
	if strings.HasPrefix(u, "@") {
		return "g:'" + strings.TrimPrefix(u, "@") + "'"
	}
	return "u:'" + u + "'"
}

// buildLockCmd wipes a subfolder's ACL, shuts out "other", then grants exactly
// the allowlist (access + default ACL) plus traverse (x) up the parent chain.
func buildLockCmd(recursiveFlag, base, target, perms string, users []string) string {
	qTarget := "'" + target + "'"
	var b strings.Builder
	fmt.Fprintf(&b, "setfacl %s-b %s && chmod %so= %s", recursiveFlag, qTarget, recursiveFlag, qTarget)
	for _, u := range users {
		qUser := "'" + u + "'"
		// Access ACL — recurse into existing contents when asked.
		fmt.Fprintf(&b, " && setfacl %s-m u:%s:%s %s", recursiveFlag, qUser, perms, qTarget)
		// Default ACL drives inheritance and is valid on directories only, so it must
		// never carry -R (setfacl errors on regular files with a default entry).
		fmt.Fprintf(&b, " && setfacl -m d:u:%s:%s %s", qUser, perms, qTarget)
		// grant traverse (x) on every ancestor up to and including the share root
		dir := filepath.Dir(target)
		for strings.HasPrefix(dir, base) && dir != base {
			fmt.Fprintf(&b, " && setfacl -m u:%s:x '%s'", qUser, dir)
			dir = filepath.Dir(dir)
		}
		fmt.Fprintf(&b, " && setfacl -m u:%s:x '%s'", qUser, base)
	}
	return b.String()
}

// buildUnlockCmd clears the private ACL, reopens the "other" class, and re-grants
// the share's valid users (mirrors sync-share-acl.sh so the folder rejoins "open").
func buildUnlockCmd(recursiveFlag, target string, validUsers []string) string {
	qTarget := "'" + target + "'"
	var b strings.Builder
	fmt.Fprintf(&b, "setfacl %s-b %s && chmod %so+rX %s", recursiveFlag, qTarget, recursiveFlag, qTarget)
	for _, u := range validUsers {
		u = strings.TrimSpace(u)
		if u == "" || !subfolderValidUser.MatchString(u) {
			continue
		}
		spec := aclSpec(u)
		fmt.Fprintf(&b, " && setfacl %s-m %s:rwX %s", recursiveFlag, spec, qTarget)
		fmt.Fprintf(&b, " && setfacl -m d:%s:rwX %s", spec, qTarget)
	}
	return b.String()
}

func (h *sharesHandlers) lockSubfolder(c *fiber.Ctx) error {
	name := c.Params("name")
	var req subfolderLockRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid request body"})
	}

	perms := strings.TrimSpace(req.Permissions)
	if perms == "" {
		perms = "rx"
	}
	if !subfolderValidPerms.MatchString(perms) {
		return c.Status(400).JSON(fiber.Map{"detail": "permissions must be an ordered subset of rwx"})
	}

	users := make([]string, 0, len(req.Users))
	for _, u := range req.Users {
		u = strings.TrimSpace(u)
		if u == "" {
			continue
		}
		if !subfolderValidUser.MatchString(u) {
			return c.Status(400).JSON(fiber.Map{"detail": "invalid username: " + u})
		}
		users = append(users, u)
	}

	basePath, targetPath, rel, err := h.resolveSubfolder(name, req.SubfolderPath)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": err.Error()})
	}
	// Locking the share root would wall off the entire share — refuse it.
	if rel == "." {
		return c.Status(400).JSON(fiber.Map{"detail": "cannot lock the share root; choose a subfolder"})
	}

	recursiveFlag := ""
	if req.Recursive {
		recursiveFlag = "-R "
	}
	cmd := buildLockCmd(recursiveFlag, basePath, targetPath, perms, users)
	if out, err := h.exec.ExecuteOutput(cmd); err != nil {
		return c.Status(500).JSON(fiber.Map{"detail": "lock failed: " + out})
	}
	h.auditSvc.Log("lock_subfolder", actor(c), "share", name, "success",
		map[string]interface{}{"path": rel, "users": users}, c.IP())
	return c.JSON(fiber.Map{"message": fmt.Sprintf("locked %q to %d user(s)", rel, len(users))})
}

func (h *sharesHandlers) unlockSubfolder(c *fiber.Ctx) error {
	name := c.Params("name")
	var req subfolderUnlockRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid request body"})
	}

	_, targetPath, rel, err := h.resolveSubfolder(name, req.SubfolderPath)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": err.Error()})
	}

	valid := []string{}
	if share, _ := h.parser().GetShare(name); share != nil {
		valid = share.ValidUsers
	}

	recursiveFlag := ""
	if req.Recursive {
		recursiveFlag = "-R "
	}
	cmd := buildUnlockCmd(recursiveFlag, targetPath, valid)
	if out, err := h.exec.ExecuteOutput(cmd); err != nil {
		return c.Status(500).JSON(fiber.Map{"detail": "unlock failed: " + out})
	}
	h.auditSvc.Log("unlock_subfolder", actor(c), "share", name, "success",
		map[string]interface{}{"path": rel}, c.IP())
	return c.JSON(fiber.Map{"message": fmt.Sprintf("unlocked %q", rel)})
}

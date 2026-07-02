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
		// access ACL + default ACL (inheritance) in one call
		cmd = fmt.Sprintf(
			"setfacl %s-m u:%s:%s,d:u:%s:%s %s",
			recursiveFlag, qUser, perms, qUser, perms, qPath,
		)
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
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		isDefault := strings.HasPrefix(line, "default:")
		line = strings.TrimPrefix(line, "default:")
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
	return c.JSON(fiber.Map{"share": name, "path": rel, "entries": entries})
}

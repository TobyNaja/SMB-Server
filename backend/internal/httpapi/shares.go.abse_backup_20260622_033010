package httpapi

import (
	"regexp"

	"smb-server/backend/internal/samba"

	"github.com/gofiber/fiber/v2"
)

// ValidSharePath allows absolute paths with safe characters only — no shell metacharacters.
var ValidSharePath = regexp.MustCompile(`^/[a-zA-Z0-9/_-]+$`)

type sharesHandlers struct {
	exec       samba.Executor
	configPath string
}

type shareCreateRequest struct {
	Name          string `json:"name"`
	Path          string `json:"path"`
	Comment       string `json:"comment"`
	Browseable    bool   `json:"browseable"`
	GuestOK       bool   `json:"guest_ok"`
	ABSE          bool   `json:"abse"`
	CreateMask    string `json:"create_mask"`
	DirectoryMask string `json:"directory_mask"`
}

type shareUpdateRequest struct {
	Comment       *string `json:"comment"`
	Browseable    *bool   `json:"browseable"`
	GuestOK       *bool   `json:"guest_ok"`
	ReadOnly      *bool   `json:"read_only"`
	ABSE          *bool   `json:"abse"`
	CreateMask    *string `json:"create_mask"`
	DirectoryMask *string `json:"directory_mask"`
}

type permissionUpdateRequest struct {
	Users          []string `json:"users"`
	PermissionType string   `json:"permission_type"`
}

type abseRequest struct {
	Enabled bool `json:"enabled"`
}

func registerSharesRoutes(g fiber.Router, exec samba.Executor, configPath string) {
	h := &sharesHandlers{exec: exec, configPath: configPath}
	// Static routes before /:name
	g.Get("/global", h.getGlobal)
	g.Patch("/global", h.updateGlobal)
	g.Get("", h.list)
	g.Post("", h.create)
	g.Patch("/:name/abse", h.toggleABSE)
	g.Post("/:name/permissions", h.updatePermissions)
	g.Get("/:name", h.get)
	g.Patch("/:name", h.update)
	g.Delete("/:name", h.delete)
}

func (h *sharesHandlers) parser() *samba.SmbConfParser {
	return samba.NewSmbConfParser(h.configPath+"/shares.conf", h.configPath+"/smb.conf")
}

func (h *sharesHandlers) getGlobal(c *fiber.Ctx) error {
	return c.JSON(h.parser().GetGlobal())
}

func (h *sharesHandlers) updateGlobal(c *fiber.Ctx) error {
	// Global config is template-managed; no-op but still reload
	h.exec.ReloadSamba()
	return c.JSON(fiber.Map{"message": "Global config is managed by template"})
}

func (h *sharesHandlers) list(c *fiber.Ctx) error {
	shares, err := h.parser().GetAllShares()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"detail": err.Error()})
	}
	return c.JSON(fiber.Map{"shares": shares})
}

func (h *sharesHandlers) create(c *fiber.Ctx) error {
	var req shareCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid request body"})
	}
	if !ValidSharePath.MatchString(req.Path) {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid path: must be absolute and contain only [a-zA-Z0-9/_-]"})
	}
	// mkdir + chmod in samba container (path is validated above — no injection risk)
	h.exec.Execute("mkdir -p " + req.Path + " && chmod 770 " + req.Path)

	p := h.parser()
	if !p.CreateShare(req.Name, req.Path, req.Comment) {
		return c.Status(400).JSON(fiber.Map{"detail": "Share already exists"})
	}
	updates := map[string]interface{}{
		"browseable":     req.Browseable,
		"guest_ok":       req.GuestOK,
		"abse":           req.ABSE,
		"create mask":    req.CreateMask,
		"directory mask": req.DirectoryMask,
	}
	p.UpdateShare(req.Name, updates)
	h.exec.ReloadSamba()
	return c.Status(201).JSON(fiber.Map{"message": "Share '" + req.Name + "' created"})
}

func (h *sharesHandlers) get(c *fiber.Ctx) error {
	name := c.Params("name")
	share, err := h.parser().GetShare(name)
	if err != nil || share == nil {
		return c.Status(404).JSON(fiber.Map{"detail": "Share not found"})
	}
	return c.JSON(share)
}

func (h *sharesHandlers) update(c *fiber.Ctx) error {
	name := c.Params("name")
	var req shareUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid request body"})
	}
	p := h.parser()
	if ok, _ := p.ShareExists(name); !ok {
		return c.Status(404).JSON(fiber.Map{"detail": "Share not found"})
	}
	updates := map[string]interface{}{}
	if req.Comment != nil {
		updates["comment"] = *req.Comment
	}
	if req.Browseable != nil {
		updates["browseable"] = *req.Browseable
	}
	if req.GuestOK != nil {
		updates["guest_ok"] = *req.GuestOK
	}
	if req.ReadOnly != nil {
		updates["read_only"] = *req.ReadOnly
	}
	if req.ABSE != nil {
		updates["abse"] = *req.ABSE
	}
	if req.CreateMask != nil {
		updates["create mask"] = *req.CreateMask
	}
	if req.DirectoryMask != nil {
		updates["directory mask"] = *req.DirectoryMask
	}
	p.UpdateShare(name, updates)
	h.exec.ReloadSamba()
	return c.JSON(fiber.Map{"message": "Share '" + name + "' updated"})
}

func (h *sharesHandlers) delete(c *fiber.Ctx) error {
	name := c.Params("name")
	h.parser().DeleteShare(name)
	h.exec.ReloadSamba()
	return c.JSON(fiber.Map{"message": "Share '" + name + "' deleted"})
}

func (h *sharesHandlers) toggleABSE(c *fiber.Ctx) error {
	name := c.Params("name")
	enabled := c.QueryBool("enabled")
	p := h.parser()
	if ok, _ := p.ShareExists(name); !ok {
		return c.Status(404).JSON(fiber.Map{"detail": "Share not found"})
	}
	p.SetShareABSE(name, enabled)
	h.exec.ReloadSamba()
	return c.JSON(fiber.Map{"message": "ABSE updated"})
}

func (h *sharesHandlers) updatePermissions(c *fiber.Ctx) error {
	name := c.Params("name")
	var req permissionUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid request body"})
	}
	p := h.parser()
	if ok, _ := p.ShareExists(name); !ok {
		return c.Status(404).JSON(fiber.Map{"detail": "Share not found"})
	}
	validTypes := map[string]func(string, []string) bool{
		"valid_users":   p.SetValidUsers,
		"write_list":    p.SetWriteList,
		"read_list":     p.SetReadList,
		"admin_users":   p.SetAdminUsers,
		"invalid_users": p.SetInvalidUsers,
	}
	setter, ok := validTypes[req.PermissionType]
	if !ok {
		return c.Status(400).JSON(fiber.Map{"detail": "Invalid permission_type: " + req.PermissionType})
	}
	setter(name, req.Users)
	h.exec.ReloadSamba()
	return c.JSON(fiber.Map{"message": "Permissions updated"})
}

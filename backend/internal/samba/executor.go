package samba

import (
	"bytes"
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)

// ExecResult holds the outcome of a container exec call.
type ExecResult struct {
	Success  bool
	ExitCode int
	Output   string
}

// UserInfo is the flat representation of a pdbedit -L entry.
type UserInfo struct {
	Username string `json:"username"`
	UID      string `json:"uid"`
	Fullname string `json:"fullname"`
	Disabled bool   `json:"disabled"`
}

// Executor is the interface all samba/ldap packages depend on.
// Implementations: dockerExecutor (live), FakeExecutor (tests).
type Executor interface {
	Execute(command string) ExecResult
	CreateUser(username, password string) ExecResult
	DeleteUser(username string) ExecResult
	SetPassword(username, password string) ExecResult
	CreateGroup(groupName string) ExecResult
	AddUserToGroup(username, groupName string) ExecResult
	RemoveUserFromGroup(username, groupName string) ExecResult
	GetUsers() []UserInfo
	GetGroups() []string
	ReloadSamba() ExecResult
}

// validUsername rejects usernames with characters outside [a-zA-Z0-9_-].
var validUsername = regexp.MustCompile(`^[a-zA-Z0-9_\-]{1,32}$`)

// dockerExecutor implements Executor against a live Docker container.
type dockerExecutor struct {
	containerName string
}

// NewDockerExecutor creates a live Docker executor for the named container.
func NewDockerExecutor(containerName string) Executor {
	return &dockerExecutor{containerName: containerName}
}

func (e *dockerExecutor) Execute(command string) ExecResult {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return ExecResult{Success: false, ExitCode: -1, Output: err.Error()}
	}
	defer cli.Close()

	execCfg := types.ExecConfig{
		Cmd:          []string{"/bin/bash", "-c", command},
		AttachStdout: true,
		AttachStderr: true,
		User:         "root",
	}
	resp, err := cli.ContainerExecCreate(ctx, e.containerName, execCfg)
	if err != nil {
		return ExecResult{Success: false, ExitCode: -1, Output: err.Error()}
	}
	attach, err := cli.ContainerExecAttach(ctx, resp.ID, types.ExecStartCheck{})
	if err != nil {
		return ExecResult{Success: false, ExitCode: -1, Output: err.Error()}
	}
	defer attach.Close()

	var buf bytes.Buffer
	if _, err := stdcopy.StdCopy(&buf, &buf, attach.Reader); err != nil {
		return ExecResult{Success: false, ExitCode: -1, Output: err.Error()}
	}

	inspect, err := cli.ContainerExecInspect(ctx, resp.ID)
	if err != nil {
		return ExecResult{Success: false, ExitCode: -1, Output: err.Error()}
	}
	return ExecResult{
		Success:  inspect.ExitCode == 0,
		ExitCode: inspect.ExitCode,
		Output:   buf.String(),
	}
}

func (e *dockerExecutor) CreateUser(username, password string) ExecResult {
	if !validUsername.MatchString(username) {
		return ExecResult{Success: false, Output: "invalid username format"}
	}
	// Ensure system user exists
	e.Execute(fmt.Sprintf(
		"id %q > /dev/null 2>&1 || useradd -m -s /usr/sbin/nologin %q",
		username, username,
	))
	// Set samba password
	escaped := strings.ReplaceAll(password, "'", "'\\''")
	return e.Execute(fmt.Sprintf(
		"printf '%s\\n%s\\n' | smbpasswd -a -s %q",
		escaped, escaped, username,
	))
}

func (e *dockerExecutor) DeleteUser(username string) ExecResult {
	e.Execute(fmt.Sprintf("smbpasswd -x %q 2>/dev/null || true", username))
	e.Execute(fmt.Sprintf("userdel -r %q 2>/dev/null || true", username))
	return ExecResult{Success: true, Output: "User " + username + " deleted"}
}

func (e *dockerExecutor) SetPassword(username, password string) ExecResult {
	escaped := strings.ReplaceAll(password, "'", "'\\''")
	return e.Execute(fmt.Sprintf(
		"printf '%s\\n%s\\n' | smbpasswd -s %q",
		escaped, escaped, username,
	))
}

func (e *dockerExecutor) CreateGroup(groupName string) ExecResult {
	e.Execute(fmt.Sprintf("groupadd %q 2>/dev/null || true", groupName))
	return ExecResult{Success: true, Output: "Group " + groupName + " created"}
}

func (e *dockerExecutor) AddUserToGroup(username, groupName string) ExecResult {
	return e.Execute(fmt.Sprintf("usermod -a -G %q %q", groupName, username))
}

func (e *dockerExecutor) RemoveUserFromGroup(username, groupName string) ExecResult {
	return e.Execute(fmt.Sprintf("gpasswd -d %q %q", username, groupName))
}

func (e *dockerExecutor) GetUsers() []UserInfo {
	result := e.Execute("pdbedit -L 2>/dev/null")
	var users []UserInfo
	if !result.Success {
		return users
	}
	for _, line := range strings.Split(result.Output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 3)
		if len(parts) < 2 {
			continue
		}
		fullname := ""
		if len(parts) == 3 {
			fullname = strings.TrimSpace(parts[2])
		}
		users = append(users, UserInfo{
			Username: parts[0],
			UID:      parts[1],
			Fullname: fullname,
		})
	}
	return users
}

var skipGroups = map[string]bool{
	"root": true, "bin": true, "sys": true, "daemon": true, "adm": true,
	"lp": true, "mail": true, "news": true, "uucp": true, "man": true,
	"proxy": true, "kmem": true, "dialout": true, "fax": true, "voice": true,
	"cdrom": true, "floppy": true, "tape": true, "sudo": true, "audio": true,
	"dip": true, "www-data": true, "backup": true, "operator": true, "list": true,
	"irc": true, "src": true, "gnats": true, "shadow": true, "utmp": true,
	"video": true, "sasl": true, "plugdev": true, "staff": true, "games": true,
	"users": true, "nogroup": true, "crontab": true, "syslog": true, "tty": true,
	"disk": true, "input": true, "netdev": true, "render": true, "sgx": true,
	"kvm": true, "messagebus": true, "samba": true, "sambashare": true,
}

func (e *dockerExecutor) GetGroups() []string {
	result := e.Execute("getent group")
	var groups []string
	if !result.Success {
		return groups
	}
	for _, line := range strings.Split(result.Output, "\n") {
		if line == "" {
			continue
		}
		name := strings.SplitN(line, ":", 2)[0]
		if !skipGroups[name] && !strings.HasPrefix(name, "_") {
			groups = append(groups, name)
		}
	}
	return groups
}

func (e *dockerExecutor) ReloadSamba() ExecResult {
	return e.Execute("smbcontrol all reload-config 2>/dev/null || pkill -HUP smbd || true")
}

// FakeExecutor is a test double for Executor.
type FakeExecutor struct {
	Calls   []string
	Results map[string]ExecResult
	Users   []UserInfo
	Groups  []string
}

func NewFakeExecutor() *FakeExecutor {
	return &FakeExecutor{
		Results: map[string]ExecResult{},
		Groups:  []string{},
	}
}

func (f *FakeExecutor) Execute(cmd string) ExecResult {
	f.Calls = append(f.Calls, cmd)
	if r, ok := f.Results[cmd]; ok {
		return r
	}
	return ExecResult{Success: true}
}

func (f *FakeExecutor) CreateUser(username, password string) ExecResult {
	f.Calls = append(f.Calls, "CreateUser:"+username)
	f.Users = append(f.Users, UserInfo{Username: username})
	return ExecResult{Success: true}
}

func (f *FakeExecutor) DeleteUser(username string) ExecResult {
	f.Calls = append(f.Calls, "DeleteUser:"+username)
	filtered := f.Users[:0]
	for _, u := range f.Users {
		if u.Username != username {
			filtered = append(filtered, u)
		}
	}
	f.Users = filtered
	return ExecResult{Success: true}
}

func (f *FakeExecutor) SetPassword(username, password string) ExecResult {
	f.Calls = append(f.Calls, "SetPassword:"+username)
	return ExecResult{Success: true}
}

func (f *FakeExecutor) CreateGroup(groupName string) ExecResult {
	f.Calls = append(f.Calls, "CreateGroup:"+groupName)
	f.Groups = append(f.Groups, groupName)
	return ExecResult{Success: true}
}

func (f *FakeExecutor) AddUserToGroup(username, groupName string) ExecResult {
	f.Calls = append(f.Calls, "AddUserToGroup:"+username+":"+groupName)
	return ExecResult{Success: true}
}

func (f *FakeExecutor) RemoveUserFromGroup(username, groupName string) ExecResult {
	f.Calls = append(f.Calls, "RemoveUserFromGroup:"+username+":"+groupName)
	return ExecResult{Success: true}
}

func (f *FakeExecutor) GetUsers() []UserInfo { return f.Users }
func (f *FakeExecutor) GetGroups() []string  { return f.Groups }

func (f *FakeExecutor) ReloadSamba() ExecResult {
	f.Calls = append(f.Calls, "ReloadSamba")
	return ExecResult{Success: true}
}

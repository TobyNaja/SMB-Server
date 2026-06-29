package port

import "smb-server/backend/internal/domain"

// ExecResult holds the outcome of a container exec call.
type ExecResult struct {
	Success  bool
	ExitCode int
	Output   string
}

// CommandRunner abstracts all exec operations against the samba container.
type CommandRunner interface {
	Execute(command string) ExecResult
	ExecuteWithInput(cmd []string, input string) ExecResult
	CreateUser(username, password string) ExecResult
	DeleteUser(username string) ExecResult
	SetPassword(username, password string) ExecResult
	CreateGroup(groupName string) ExecResult
	AddUserToGroup(username, groupName string) ExecResult
	RemoveUserFromGroup(username, groupName string) ExecResult
	GetUsers() []domain.User
	GetGroups() []string
	ReloadSamba() ExecResult
}

package fake

import (
    "strings"
    "smb-server/backend/internal/domain"
    "smb-server/backend/internal/port"
)

type CommandRunner struct {
    Calls   []string
    Results map[string]port.ExecResult
    users   []domain.User
    groups  []string
}

func NewCommandRunner() *CommandRunner {
    return &CommandRunner{Results: map[string]port.ExecResult{}, groups: []string{}}
}

func (f *CommandRunner) Execute(cmd string) port.ExecResult {
    f.Calls = append(f.Calls, cmd)
    if r, ok := f.Results[cmd]; ok { return r }
    return port.ExecResult{Success: true}
}

func (f *CommandRunner) ExecuteWithInput(cmd []string, _ string) port.ExecResult {
    f.Calls = append(f.Calls, "ExecuteWithInput:"+strings.Join(cmd, " "))
    return port.ExecResult{Success: true}
}

func (f *CommandRunner) CreateUser(username, _ string) port.ExecResult {
    f.Calls = append(f.Calls, "CreateUser:"+username)
    f.users = append(f.users, domain.User{Username: username})
    return port.ExecResult{Success: true}
}

func (f *CommandRunner) DeleteUser(username string) port.ExecResult {
    f.Calls = append(f.Calls, "DeleteUser:"+username)
    out := f.users[:0]
    for _, u := range f.users {
        if u.Username != username { out = append(out, u) }
    }
    f.users = out
    return port.ExecResult{Success: true}
}

func (f *CommandRunner) SetPassword(username, _ string) port.ExecResult {
    f.Calls = append(f.Calls, "SetPassword:"+username)
    return port.ExecResult{Success: true}
}

func (f *CommandRunner) CreateGroup(name string) port.ExecResult {
    f.Calls = append(f.Calls, "CreateGroup:"+name)
    f.groups = append(f.groups, name)
    return port.ExecResult{Success: true}
}

func (f *CommandRunner) AddUserToGroup(username, group string) port.ExecResult {
    f.Calls = append(f.Calls, "AddUserToGroup:"+username+":"+group)
    return port.ExecResult{Success: true}
}

func (f *CommandRunner) RemoveUserFromGroup(username, group string) port.ExecResult {
    f.Calls = append(f.Calls, "RemoveUserFromGroup:"+username+":"+group)
    return port.ExecResult{Success: true}
}

func (f *CommandRunner) GetUsers() []domain.User  { return f.users }
func (f *CommandRunner) GetGroups() []string       { return f.groups }

func (f *CommandRunner) ReloadSamba() port.ExecResult {
    f.Calls = append(f.Calls, "ReloadSamba")
    return port.ExecResult{Success: true}
}

var _ port.CommandRunner = (*CommandRunner)(nil)

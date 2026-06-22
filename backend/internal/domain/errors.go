package domain

import "errors"

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
	ErrInvalidPath   = errors.New("invalid path")
	ErrLastAdmin     = errors.New("cannot delete last admin")
	ErrSelfDelete    = errors.New("cannot delete yourself")
)

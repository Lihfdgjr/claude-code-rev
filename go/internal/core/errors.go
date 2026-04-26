package core

import "errors"

var (
	ErrToolNotFound     = errors.New("tool not found")
	ErrCommandNotFound  = errors.New("command not found")
	ErrPermissionDenied = errors.New("permission denied")
	ErrAPIInvalid       = errors.New("api response invalid")
	ErrCancelled        = errors.New("cancelled")
	ErrConfigMissing    = errors.New("config missing")
)

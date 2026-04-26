package core

import "context"

type PermissionDecision int

const (
	PermissionAllow PermissionDecision = iota
	PermissionDeny
	PermissionAsk
)

type PermissionRequest struct {
	Tool  string
	Input []byte
}

type PermissionResponse struct {
	Decision PermissionDecision
	Remember bool
}

type PermissionGate interface {
	Check(ctx context.Context, req PermissionRequest) (PermissionDecision, string)
	AllowRuntime(tool string)
}

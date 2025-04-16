package permission

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kujtimiihoxha/opencode/internal/pubsub"
)

var ErrorPermissionDenied = errors.New("permission denied")

type CreatePermissionRequest struct {
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Params      any    `json:"params"`
	Path        string `json:"path"`
}

type PermissionRequest struct {
	ID          string `json:"id"`
	SessionID   string `json:"session_id"`
	ToolName    string `json:"tool_name"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Params      any    `json:"params"`
	Path        string `json:"path"`
}

type Service interface {
	pubsub.Suscriber[PermissionRequest]
	GrantPersistant(permission PermissionRequest)
	Grant(permission PermissionRequest)
	Deny(permission PermissionRequest)
	Request(opts CreatePermissionRequest) bool
}

type permissionService struct {
	*pubsub.Broker[PermissionRequest]

	sessionPermissions []PermissionRequest
	pendingRequests    sync.Map
}

func (s *permissionService) GrantPersistant(permission PermissionRequest) {
	respCh, ok := s.pendingRequests.Load(permission.ID)
	if ok {
		respCh.(chan bool) <- true
	}
	s.sessionPermissions = append(s.sessionPermissions, permission)
}

func (s *permissionService) Grant(permission PermissionRequest) {
	respCh, ok := s.pendingRequests.Load(permission.ID)
	if ok {
		respCh.(chan bool) <- true
	}
}

func (s *permissionService) Deny(permission PermissionRequest) {
	respCh, ok := s.pendingRequests.Load(permission.ID)
	if ok {
		respCh.(chan bool) <- false
	}
}

func (s *permissionService) Request(opts CreatePermissionRequest) bool {
	permission := PermissionRequest{
		ID:          uuid.New().String(),
		Path:        opts.Path,
		ToolName:    opts.ToolName,
		Description: opts.Description,
		Action:      opts.Action,
		Params:      opts.Params,
	}

	for _, p := range s.sessionPermissions {
		if p.ToolName == permission.ToolName && p.Action == permission.Action {
			return true
		}
	}

	respCh := make(chan bool, 1)

	s.pendingRequests.Store(permission.ID, respCh)
	defer s.pendingRequests.Delete(permission.ID)

	s.Publish(pubsub.CreatedEvent, permission)

	// Wait for the response with a timeout
	select {
	case resp := <-respCh:
		return resp
	case <-time.After(10 * time.Minute):
		return false
	}
}

func NewPermissionService() Service {
	return &permissionService{
		Broker:             pubsub.NewBroker[PermissionRequest](),
		sessionPermissions: make([]PermissionRequest, 0),
	}
}

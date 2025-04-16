package tools

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/kujtimiihoxha/termai/internal/history"
	"github.com/kujtimiihoxha/termai/internal/permission"
	"github.com/kujtimiihoxha/termai/internal/pubsub"
)

// Mock permission service for testing
type mockPermissionService struct {
	*pubsub.Broker[permission.PermissionRequest]
	allow bool
}

func (m *mockPermissionService) GrantPersistant(permission permission.PermissionRequest) {
	// Not needed for tests
}

func (m *mockPermissionService) Grant(permission permission.PermissionRequest) {
	// Not needed for tests
}

func (m *mockPermissionService) Deny(permission permission.PermissionRequest) {
	// Not needed for tests
}

func (m *mockPermissionService) Request(opts permission.CreatePermissionRequest) bool {
	return m.allow
}

func newMockPermissionService(allow bool) permission.Service {
	return &mockPermissionService{
		Broker: pubsub.NewBroker[permission.PermissionRequest](),
		allow:  allow,
	}
}

type mockFileHistoryService struct {
	*pubsub.Broker[history.File]
	files     map[string]history.File // ID -> File
	timeNow   func() int64
}

// Create implements history.Service.
func (m *mockFileHistoryService) Create(ctx context.Context, sessionID string, path string, content string) (history.File, error) {
	return m.createWithVersion(ctx, sessionID, path, content, history.InitialVersion)
}

// CreateVersion implements history.Service.
func (m *mockFileHistoryService) CreateVersion(ctx context.Context, sessionID string, path string, content string) (history.File, error) {
	var files []history.File
	for _, file := range m.files {
		if file.Path == path {
			files = append(files, file)
		}
	}

	if len(files) == 0 {
		// No previous versions, create initial
		return m.Create(ctx, sessionID, path, content)
	}

	// Sort files by CreatedAt in descending order
	sort.Slice(files, func(i, j int) bool {
		return files[i].CreatedAt > files[j].CreatedAt
	})

	// Get the latest version
	latestFile := files[0]
	latestVersion := latestFile.Version

	// Generate the next version
	var nextVersion string
	if latestVersion == history.InitialVersion {
		nextVersion = "v1"
	} else if strings.HasPrefix(latestVersion, "v") {
		versionNum, err := strconv.Atoi(latestVersion[1:])
		if err != nil {
			// If we can't parse the version, just use a timestamp-based version
			nextVersion = fmt.Sprintf("v%d", latestFile.CreatedAt)
		} else {
			nextVersion = fmt.Sprintf("v%d", versionNum+1)
		}
	} else {
		// If the version format is unexpected, use a timestamp-based version
		nextVersion = fmt.Sprintf("v%d", latestFile.CreatedAt)
	}

	return m.createWithVersion(ctx, sessionID, path, content, nextVersion)
}

func (m *mockFileHistoryService) createWithVersion(_ context.Context, sessionID, path, content, version string) (history.File, error) {
	now := m.timeNow()
	file := history.File{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Path:      path,
		Content:   content,
		Version:   version,
		CreatedAt: now,
		UpdatedAt: now,
	}

	m.files[file.ID] = file
	m.Publish(pubsub.CreatedEvent, file)
	return file, nil
}

// Delete implements history.Service.
func (m *mockFileHistoryService) Delete(ctx context.Context, id string) error {
	file, ok := m.files[id]
	if !ok {
		return fmt.Errorf("file not found: %s", id)
	}

	delete(m.files, id)
	m.Publish(pubsub.DeletedEvent, file)
	return nil
}

// DeleteSessionFiles implements history.Service.
func (m *mockFileHistoryService) DeleteSessionFiles(ctx context.Context, sessionID string) error {
	files, err := m.ListBySession(ctx, sessionID)
	if err != nil {
		return err
	}

	for _, file := range files {
		err = m.Delete(ctx, file.ID)
		if err != nil {
			return err
		}
	}

	return nil
}

// Get implements history.Service.
func (m *mockFileHistoryService) Get(ctx context.Context, id string) (history.File, error) {
	file, ok := m.files[id]
	if !ok {
		return history.File{}, fmt.Errorf("file not found: %s", id)
	}
	return file, nil
}

// GetByPathAndSession implements history.Service.
func (m *mockFileHistoryService) GetByPathAndSession(ctx context.Context, path string, sessionID string) (history.File, error) {
	var latestFile history.File
	var found bool
	var latestTime int64

	for _, file := range m.files {
		if file.Path == path && file.SessionID == sessionID {
			if !found || file.CreatedAt > latestTime {
				latestFile = file
				latestTime = file.CreatedAt
				found = true
			}
		}
	}

	if !found {
		return history.File{}, fmt.Errorf("file not found: %s for session %s", path, sessionID)
	}
	return latestFile, nil
}

// ListBySession implements history.Service.
func (m *mockFileHistoryService) ListBySession(ctx context.Context, sessionID string) ([]history.File, error) {
	var files []history.File
	for _, file := range m.files {
		if file.SessionID == sessionID {
			files = append(files, file)
		}
	}

	// Sort by CreatedAt in descending order
	sort.Slice(files, func(i, j int) bool {
		return files[i].CreatedAt > files[j].CreatedAt
	})

	return files, nil
}

// ListLatestSessionFiles implements history.Service.
func (m *mockFileHistoryService) ListLatestSessionFiles(ctx context.Context, sessionID string) ([]history.File, error) {
	// Map to track the latest file for each path
	latestFiles := make(map[string]history.File)
	
	for _, file := range m.files {
		if file.SessionID == sessionID {
			existing, ok := latestFiles[file.Path]
			if !ok || file.CreatedAt > existing.CreatedAt {
				latestFiles[file.Path] = file
			}
		}
	}

	// Convert map to slice
	var result []history.File
	for _, file := range latestFiles {
		result = append(result, file)
	}

	// Sort by CreatedAt in descending order
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt > result[j].CreatedAt
	})

	return result, nil
}

// Subscribe implements history.Service.
func (m *mockFileHistoryService) Subscribe(ctx context.Context) <-chan pubsub.Event[history.File] {
	return m.Broker.Subscribe(ctx)
}

// Update implements history.Service.
func (m *mockFileHistoryService) Update(ctx context.Context, file history.File) (history.File, error) {
	_, ok := m.files[file.ID]
	if !ok {
		return history.File{}, fmt.Errorf("file not found: %s", file.ID)
	}

	file.UpdatedAt = m.timeNow()
	m.files[file.ID] = file
	m.Publish(pubsub.UpdatedEvent, file)
	return file, nil
}

func newMockFileHistoryService() history.Service {
	return &mockFileHistoryService{
		Broker:   pubsub.NewBroker[history.File](),
		files:    make(map[string]history.File),
		timeNow:  func() int64 { return time.Now().Unix() },
	}
}

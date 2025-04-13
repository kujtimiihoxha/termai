package history

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/kujtimiihoxha/termai/internal/db"
	"github.com/kujtimiihoxha/termai/internal/pubsub"
)

const (
	InitialVersion = "initial"
)

type File struct {
	ID        string
	SessionID string
	Path      string
	Content   string
	Version   string
	CreatedAt int64
	UpdatedAt int64
}

type Service interface {
	pubsub.Suscriber[File]
	Create(sessionID, path, content string) (File, error)
	CreateVersion(sessionID, path, content string) (File, error)
	Get(id string) (File, error)
	GetByPathAndSession(path, sessionID string) (File, error)
	ListBySession(sessionID string) ([]File, error)
	ListLatestSessionFiles(sessionID string) ([]File, error)
	Update(file File) (File, error)
	Delete(id string) error
	DeleteSessionFiles(sessionID string) error
}

type service struct {
	*pubsub.Broker[File]
	q   db.Querier
	ctx context.Context
}

func NewService(ctx context.Context, q db.Querier) Service {
	return &service{
		Broker: pubsub.NewBroker[File](),
		q:      q,
		ctx:    ctx,
	}
}

func (s *service) Create(sessionID, path, content string) (File, error) {
	return s.createWithVersion(sessionID, path, content, InitialVersion)
}

func (s *service) CreateVersion(sessionID, path, content string) (File, error) {
	// Get the latest version for this path
	files, err := s.q.ListFilesByPath(s.ctx, path)
	if err != nil {
		return File{}, err
	}

	if len(files) == 0 {
		// No previous versions, create initial
		return s.Create(sessionID, path, content)
	}

	// Get the latest version
	latestFile := files[0] // Files are ordered by created_at DESC
	latestVersion := latestFile.Version

	// Generate the next version
	var nextVersion string
	if latestVersion == InitialVersion {
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

	return s.createWithVersion(sessionID, path, content, nextVersion)
}

func (s *service) createWithVersion(sessionID, path, content, version string) (File, error) {
	dbFile, err := s.q.CreateFile(s.ctx, db.CreateFileParams{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Path:      path,
		Content:   content,
		Version:   version,
	})
	if err != nil {
		return File{}, err
	}
	file := s.fromDBItem(dbFile)
	s.Publish(pubsub.CreatedEvent, file)
	return file, nil
}

func (s *service) Get(id string) (File, error) {
	dbFile, err := s.q.GetFile(s.ctx, id)
	if err != nil {
		return File{}, err
	}
	return s.fromDBItem(dbFile), nil
}

func (s *service) GetByPathAndSession(path, sessionID string) (File, error) {
	dbFile, err := s.q.GetFileByPathAndSession(s.ctx, db.GetFileByPathAndSessionParams{
		Path:      path,
		SessionID: sessionID,
	})
	if err != nil {
		return File{}, err
	}
	return s.fromDBItem(dbFile), nil
}

func (s *service) ListBySession(sessionID string) ([]File, error) {
	dbFiles, err := s.q.ListFilesBySession(s.ctx, sessionID)
	if err != nil {
		return nil, err
	}
	files := make([]File, len(dbFiles))
	for i, dbFile := range dbFiles {
		files[i] = s.fromDBItem(dbFile)
	}
	return files, nil
}

func (s *service) ListLatestSessionFiles(sessionID string) ([]File, error) {
	dbFiles, err := s.q.ListLatestSessionFiles(s.ctx, sessionID)
	if err != nil {
		return nil, err
	}
	files := make([]File, len(dbFiles))
	for i, dbFile := range dbFiles {
		files[i] = s.fromDBItem(dbFile)
	}
	return files, nil
}

func (s *service) Update(file File) (File, error) {
	dbFile, err := s.q.UpdateFile(s.ctx, db.UpdateFileParams{
		ID:      file.ID,
		Content: file.Content,
		Version: file.Version,
	})
	if err != nil {
		return File{}, err
	}
	updatedFile := s.fromDBItem(dbFile)
	s.Publish(pubsub.UpdatedEvent, updatedFile)
	return updatedFile, nil
}

func (s *service) Delete(id string) error {
	file, err := s.Get(id)
	if err != nil {
		return err
	}
	err = s.q.DeleteFile(s.ctx, id)
	if err != nil {
		return err
	}
	s.Publish(pubsub.DeletedEvent, file)
	return nil
}

func (s *service) DeleteSessionFiles(sessionID string) error {
	files, err := s.ListBySession(sessionID)
	if err != nil {
		return err
	}
	for _, file := range files {
		err = s.Delete(file.ID)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *service) fromDBItem(item db.File) File {
	return File{
		ID:        item.ID,
		SessionID: item.SessionID,
		Path:      item.Path,
		Content:   item.Content,
		Version:   item.Version,
		CreatedAt: item.CreatedAt,
		UpdatedAt: item.UpdatedAt,
	}
}


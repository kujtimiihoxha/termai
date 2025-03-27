package message

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/kujtimiihoxha/termai/internal/db"
	"github.com/kujtimiihoxha/termai/internal/pubsub"
)

type MessageRole string

const (
	Assistant MessageRole = "assistant"
	User      MessageRole = "user"
	System    MessageRole = "system"
	Tool      MessageRole = "tool"
)

type ToolResult struct {
	ToolCallID string
	Content    string
	IsError    bool
	// TODO: support for images
}

type ToolCall struct {
	ID    string
	Name  string
	Input string
	Type  string
}

type Message struct {
	ID        string
	SessionID string

	// NEW
	Role    MessageRole
	Content string

	ToolResults []ToolResult
	ToolCalls   []ToolCall
	CreatedAt   int64
	UpdatedAt   int64
}

type CreateMessageParams struct {
	Role        MessageRole
	Content     string
	ToolCalls   []ToolCall
	ToolResults []ToolResult
}

type Service interface {
	pubsub.Suscriber[Message]
	Create(sessionID string, params CreateMessageParams) (Message, error)
	Update(message Message) error
	Get(id string) (Message, error)
	List(sessionID string) ([]Message, error)
	Delete(id string) error
	DeleteSessionMessages(sessionID string) error
}

type service struct {
	*pubsub.Broker[Message]
	q   db.Querier
	ctx context.Context
}

func (s *service) Delete(id string) error {
	message, err := s.Get(id)
	if err != nil {
		return err
	}
	err = s.q.DeleteMessage(s.ctx, message.ID)
	if err != nil {
		return err
	}
	s.Publish(pubsub.DeletedEvent, message)
	return nil
}

func (s *service) Create(sessionID string, params CreateMessageParams) (Message, error) {
	toolCallsStr, err := json.Marshal(params.ToolCalls)
	if err != nil {
		return Message{}, err
	}
	toolResultsStr, err := json.Marshal(params.ToolResults)
	if err != nil {
		return Message{}, err
	}
	dbMessage, err := s.q.CreateMessage(s.ctx, db.CreateMessageParams{
		ID:          uuid.New().String(),
		SessionID:   sessionID,
		Role:        string(params.Role),
		Content:     params.Content,
		ToolCalls:   sql.NullString{String: string(toolCallsStr), Valid: true},
		ToolResults: sql.NullString{String: string(toolResultsStr), Valid: true},
	})
	if err != nil {
		return Message{}, err
	}
	message, err := s.fromDBItem(dbMessage)
	if err != nil {
		return Message{}, err
	}
	s.Publish(pubsub.CreatedEvent, message)
	return message, nil
}

func (s *service) DeleteSessionMessages(sessionID string) error {
	messages, err := s.List(sessionID)
	if err != nil {
		return err
	}
	for _, message := range messages {
		if message.SessionID == sessionID {
			err = s.Delete(message.ID)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *service) Update(message Message) error {
	toolCallsStr, err := json.Marshal(message.ToolCalls)
	if err != nil {
		return err
	}
	toolResultsStr, err := json.Marshal(message.ToolResults)
	if err != nil {
		return err
	}
	err = s.q.UpdateMessage(s.ctx, db.UpdateMessageParams{
		ID:          message.ID,
		Content:     message.Content,
		ToolCalls:   sql.NullString{String: string(toolCallsStr), Valid: true},
		ToolResults: sql.NullString{String: string(toolResultsStr), Valid: true},
	})
	if err != nil {
		return err
	}
	s.Publish(pubsub.UpdatedEvent, message)
	return nil
}

func (s *service) Get(id string) (Message, error) {
	dbMessage, err := s.q.GetMessage(s.ctx, id)
	if err != nil {
		return Message{}, err
	}
	return s.fromDBItem(dbMessage)
}

func (s *service) List(sessionID string) ([]Message, error) {
	dbMessages, err := s.q.ListMessagesBySession(s.ctx, sessionID)
	if err != nil {
		return nil, err
	}
	messages := make([]Message, len(dbMessages))
	for i, dbMessage := range dbMessages {
		messages[i], err = s.fromDBItem(dbMessage)
		if err != nil {
			return nil, err
		}
	}
	return messages, nil
}

func (s *service) fromDBItem(item db.Message) (Message, error) {
	toolCalls := make([]ToolCall, 0)
	if item.ToolCalls.Valid {
		err := json.Unmarshal([]byte(item.ToolCalls.String), &toolCalls)
		if err != nil {
			return Message{}, err
		}
	}

	toolResults := make([]ToolResult, 0)
	if item.ToolResults.Valid {
		err := json.Unmarshal([]byte(item.ToolResults.String), &toolResults)
		if err != nil {
			return Message{}, err
		}
	}

	return Message{
		ID:          item.ID,
		SessionID:   item.SessionID,
		Role:        MessageRole(item.Role),
		Content:     item.Content,
		ToolCalls:   toolCalls,
		ToolResults: toolResults,
		CreatedAt:   item.CreatedAt,
		UpdatedAt:   item.UpdatedAt,
	}, nil
}

func NewService(ctx context.Context, q db.Querier) Service {
	return &service{
		Broker: pubsub.NewBroker[Message](),
		q:      q,
		ctx:    ctx,
	}
}

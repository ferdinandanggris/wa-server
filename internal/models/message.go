package models

import (
	"context"
	"time"
)

// Message represents a WhatsApp message within a conversation.
type Message struct {
	ID               string     `json:"id"`
	ConversationID   string     `json:"conversation_id"`
	MessageID        string     `json:"message_id"`
	Direction        string     `json:"direction"`
	MessageType      string     `json:"message_type"`
	Content          string     `json:"content,omitempty"`
	TemplateID       string     `json:"template_id,omitempty"`
	TemplateParams   string     `json:"template_params,omitempty"`
	LanguageCode     string     `json:"language_code,omitempty"`
	MediaURL         string     `json:"media_url,omitempty"`
	Status           string     `json:"status"`
	WAStatus         string     `json:"wa_status,omitempty"`
	ContextMessageID string     `json:"context_message_id,omitempty"`
	IdempotencyKey   string     `json:"idempotency_key,omitempty"`
	MessageTimestamp int64      `json:"message_timestamp,omitempty"`
	SentAt           *time.Time `json:"sent_at,omitempty"`
	DeliveredAt      *time.Time `json:"delivered_at,omitempty"`
	ReadAt           *time.Time `json:"read_at,omitempty"`
	ErrorMessage     string     `json:"error_message,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
}

// MessageDirection indicates whether a message is inbound or outbound.
type MessageDirection string

const (
	MessageDirectionInbound  MessageDirection = "inbound"
	MessageDirectionOutbound MessageDirection = "outbound"
)

// MessageType represents the content type of a WhatsApp message.
type MessageType string

const (
	MessageTypeText     MessageType = "text"
	MessageTypeImage    MessageType = "image"
	MessageTypeVideo    MessageType = "video"
	MessageTypeDocument MessageType = "document"
	MessageTypeTemplate MessageType = "template"
	MessageTypeAudio    MessageType = "audio"
	MessageTypeSticker  MessageType = "sticker"
)

// MessageStatus represents the delivery status of a message.
type MessageStatus string

const (
	MessageStatusPending   MessageStatus = "pending"
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

type ReplyContext struct {
	Content   string
	Direction string
	Type      string
}

// MessageRepository defines persistence operations for messages.
type MessageRepository interface {
	Create(ctx context.Context, msg *Message) error
	GetByID(ctx context.Context, id string) (*Message, error)
	GetByMessageID(ctx context.Context, messageID string) (*Message, error)
	GetByIdempotencyKey(ctx context.Context, key string) (*Message, error)
	GetByConversationID(ctx context.Context, convID string, limit, offset int) ([]Message, error)
	GetReplyContext(ctx context.Context, id string) (*ReplyContext, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateDeliveryStatus(ctx context.Context, id, status string, timestamp time.Time) error
	UpdateWAMessageID(ctx context.Context, id, waMessageID string) error
	SetFailed(ctx context.Context, id, errMsg string) error
}

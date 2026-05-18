package models

import (
	"context"
	"time"
)

type Message struct {
	ID             string     `json:"id"`
	ConversationID string     `json:"conversation_id"`
	MessageID      string     `json:"message_id"`
	Direction      string     `json:"direction"`
	MessageType    string     `json:"message_type"`
	Content        string     `json:"content,omitempty"`
	TemplateID     string     `json:"template_id,omitempty"`
	TemplateParams string     `json:"template_params,omitempty"`
	MediaURL       string     `json:"media_url,omitempty"`
	Status         string     `json:"status"`
	WAStatus       string     `json:"wa_status,omitempty"`
	SentAt         *time.Time `json:"sent_at,omitempty"`
	DeliveredAt    *time.Time `json:"delivered_at,omitempty"`
	ReadAt         *time.Time `json:"read_at,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

type MessageDirection string

const (
	MessageDirectionInbound  MessageDirection = "inbound"
	MessageDirectionOutbound MessageDirection = "outbound"
)

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

type MessageStatus string

const (
	MessageStatusPending   MessageStatus = "pending"
	MessageStatusSent      MessageStatus = "sent"
	MessageStatusDelivered MessageStatus = "delivered"
	MessageStatusRead      MessageStatus = "read"
	MessageStatusFailed    MessageStatus = "failed"
)

type MessageRepository interface {
	Create(ctx context.Context, msg *Message) error
	GetByID(ctx context.Context, id string) (*Message, error)
	GetByMessageID(ctx context.Context, messageID string) (*Message, error)
	GetByConversationID(ctx context.Context, convID string, limit, offset int) ([]Message, error)
	UpdateStatus(ctx context.Context, id string, status string) error
	UpdateWAMessageID(ctx context.Context, id, waMessageID string) error
}

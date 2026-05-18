package repository

import (
	"context"
	"testing"
	"time"

	"github.com/wa-server/internal/models"
)

func TestNullJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  interface{}
	}{
		{
			name:  "empty string returns nil",
			input: "",
			want:  nil,
		},
		{
			name:  "null string returns nil",
			input: "null",
			want:  nil,
		},
		{
			name:  "valid JSON returns string",
			input: `{"key":"value"}`,
			want:  `{"key":"value"}`,
		},
		{
			name:  "plain text returns as is",
			input: "plain text",
			want:  "plain text",
		},
		{
			name:  "JSON array",
			input: `["a","b","c"]`,
			want:  `["a","b","c"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nullJSON(tt.input)
			if got != tt.want {
				t.Errorf("nullJSON(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

func TestGenerateUUID(t *testing.T) {
	for i := 0; i < 3; i++ {
		t.Run("generate uuid", func(t *testing.T) {
			got := generateUUID()
			if len(got) != 36 {
				t.Errorf("generateUUID() length = %d, want 36", len(got))
			}
		})
	}
}

func TestMessageModel(t *testing.T) {
	tests := []struct {
		name string
		fn   func(t *testing.T)
	}{
		{
			name: "message fields set correctly",
			fn: func(t *testing.T) {
				msg := &models.Message{
					ID:             "test-id",
					ConversationID: "conv-id",
					MessageID:      "msg-id",
					Direction:      "inbound",
					MessageType:    "text",
					Content:        "Hello",
					Status:         "delivered",
					CreatedAt:      time.Now(),
				}

				if msg.ID != "test-id" {
					t.Errorf("ID = %v, want test-id", msg.ID)
				}
				if msg.Direction != "inbound" {
					t.Errorf("Direction = %v, want inbound", msg.Direction)
				}
			},
		},
		{
			name: "message direction constants",
			fn: func(t *testing.T) {
				if models.MessageDirectionInbound != "inbound" {
					t.Errorf("MessageDirectionInbound = %v, want inbound", models.MessageDirectionInbound)
				}
				if models.MessageDirectionOutbound != "outbound" {
					t.Errorf("MessageDirectionOutbound = %v, want outbound", models.MessageDirectionOutbound)
				}
			},
		},
		{
			name: "message status constants",
			fn: func(t *testing.T) {
				statuses := []string{
					string(models.MessageStatusPending),
					string(models.MessageStatusSent),
					string(models.MessageStatusDelivered),
					string(models.MessageStatusRead),
					string(models.MessageStatusFailed),
				}

				expected := []string{"pending", "sent", "delivered", "read", "failed"}
				for i, status := range statuses {
					if status != expected[i] {
						t.Errorf("status[%d] = %v, want %v", i, status, expected[i])
					}
				}
			},
		},
		{
			name: "conversation status constants",
			fn: func(t *testing.T) {
				statuses := []string{
					string(models.ConversationStatusOpen),
					string(models.ConversationStatusAssigned),
					string(models.ConversationStatusClosed),
					string(models.ConversationStatusEscalated),
				}

				expected := []string{"open", "assigned", "closed", "escalated"}
				for i, status := range statuses {
					if status != expected[i] {
						t.Errorf("status[%d] = %v, want %v", i, status, expected[i])
					}
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.fn)
	}
}

func TestContextCancellation(t *testing.T) {
	tests := []struct {
		name    string
		ctxFunc func() (context.Context, context.CancelFunc)
		wantErr bool
	}{
		{
			name: "context not cancelled",
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.Background(), func() {}
			},
			wantErr: false,
		},
		{
			name: "context with timeout - not expired",
			ctxFunc: func() (context.Context, context.CancelFunc) {
				return context.WithTimeout(context.Background(), time.Hour)
			},
			wantErr: false,
		},
		{
			name: "context already cancelled",
			ctxFunc: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx, func() {}
			},
			wantErr: true,
		},
		{
			name: "context with deadline already passed",
			ctxFunc: func() (context.Context, context.CancelFunc) {
				ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Hour))
				return ctx, cancel
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := tt.ctxFunc()
			defer cancel()

			err := ctx.Err()
			gotErr := err != nil

			if gotErr != tt.wantErr {
				t.Errorf("ctx.Err() = %v, wantErr = %v", err, tt.wantErr)
			}
		})
	}
}

func TestMessageFilter(t *testing.T) {
	tests := []struct {
		name   string
		filter MessageFilter
	}{
		{
			name:   "empty filter",
			filter: MessageFilter{},
		},
		{
			name: "filter with company ID",
			filter: MessageFilter{
				CompanyID: "11111111-1111-1111-1111-111111111111",
			},
		},
		{
			name: "filter with conversation ID",
			filter: MessageFilter{
				ConversationID: "22222222-2222-2222-2222-222222222222",
			},
		},
		{
			name: "filter with status",
			filter: MessageFilter{
				Status: "delivered",
			},
		},
		{
			name: "filter with direction",
			filter: MessageFilter{
				Direction: "outbound",
			},
		},
		{
			name: "filter with time range",
			filter: MessageFilter{
				From: time.Now().Add(-time.Hour),
				To:   time.Now(),
			},
		},
		{
			name: "filter with all fields",
			filter: MessageFilter{
				CompanyID:      "11111111-1111-1111-1111-111111111111",
				ConversationID: "22222222-2222-2222-2222-222222222222",
				Status:         "sent",
				Direction:      "inbound",
				From:           time.Now().Add(-24 * time.Hour),
				To:             time.Now(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.filter
		})
	}
}

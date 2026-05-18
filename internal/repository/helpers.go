// Package repository provides database access layer for the WhatsApp gateway.
// It implements the repository pattern with PostgreSQL and handles tenant isolation.
package repository

import "github.com/google/uuid"

func generateUUID() string {
	return uuid.New().String()
}

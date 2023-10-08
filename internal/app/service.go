package app

import (
	"context"
	"errors"
	"time"
)

// ErrBlocked reports if service is blocked.
var ErrBlocked = errors.New("blocked")

// service - пустая заглушка для имплементации serviceI.
type service struct{}

// NewService - конструктор для пустой имплементации serviceI.
func NewService() *service {
	return &service{}
}

// GetLimits - возвращает всегда статические данные.
func (s *service) GetLimits() (n uint64, p time.Duration) {
	return 10, 10 * time.Second
}

// Process - имитирует бурную деятельность, возвращает пустую ошибку.
func (s *service) Process(_ context.Context, _ Batch) error {
	return nil
}

// Batch is a batch of items.
type Batch []Item

// Item is some abstract item.
type Item struct{}

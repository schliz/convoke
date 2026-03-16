package model

import "time"

type User struct {
	ID          int64
	Email       string
	DisplayName string
	Groups      []string
	IsAdmin     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

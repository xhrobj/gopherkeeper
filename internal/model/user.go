package model

import (
	"time"
)

// User представляет зарегистрированного пользователя GophKeeper.
type User struct {
	ID        int64
	Login     string
	CreatedAt time.Time
}

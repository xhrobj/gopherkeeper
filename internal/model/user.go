package model

import (
	"errors"
	"time"
)

// ErrLoginAlreadyExists сообщает, что пользователь с таким логином уже существует.
var ErrLoginAlreadyExists = errors.New("login already exists")

// User представляет зарегистрированного пользователя GophKeeper.
type User struct {
	ID        int64
	Login     string
	CreatedAt time.Time
}

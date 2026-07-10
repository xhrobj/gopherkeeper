package model

import (
	"errors"
	"time"
)

var (
	// ErrLoginAlreadyExists сообщает, что пользователь с таким логином уже существует.
	ErrLoginAlreadyExists = errors.New("login already exists")

	// ErrUserNotFound сообщает, что пользователь не найден.
	ErrUserNotFound = errors.New("user not found")
)

// User представляет зарегистрированного пользователя GophKeeper.
type User struct {
	// ID содержит внутренний идентификатор пользователя.
	ID int64

	// Login содержит канонический login пользователя.
	Login string

	// CreatedAt содержит время регистрации пользователя в UTC.
	CreatedAt time.Time
}

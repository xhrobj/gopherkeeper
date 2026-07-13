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

	// ErrInvalidCredentials сообщает, что login или password не прошли проверку.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrUnauthorized сообщает, что запрос не авторизован.
	ErrUnauthorized = errors.New("unauthorized")
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

// Authentication содержит transport-neutral результат успешной аутентификации.
type Authentication struct {
	// AccessToken содержит bearer token для авторизованных online-запросов.
	AccessToken string

	// ExpiresAt содержит время истечения срока действия token'а.
	ExpiresAt time.Time

	// User содержит данные аутентифицированного пользователя.
	User User
}

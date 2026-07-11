package model

import (
	"errors"
	"strings"
	"unicode/utf8"
)

var (
	// ErrInvalidCardPayload сообщает, что card payload некорректен.
	ErrInvalidCardPayload = errors.New("invalid card payload")
)

// CardPayload содержит приватные данные банковской карты.
type CardPayload struct {
	// Number содержит номер банковской карты.
	Number string `json:"number"`

	// Cardholder содержит необязательное имя владельца карты.
	Cardholder string `json:"cardholder,omitempty"`

	// ExpiryMonth содержит необязательный месяц окончания срока действия карты.
	ExpiryMonth *int `json:"expiry_month,omitempty"`

	// ExpiryYear содержит необязательный год окончания срока действия карты.
	ExpiryYear *int `json:"expiry_year,omitempty"`

	// CVV содержит необязательный проверочный код карты.
	CVV string `json:"cvv,omitempty"`

	// Metadata содержит необязательную произвольную текстовую метаинформацию.
	Metadata string `json:"metadata,omitempty"`
}

// Validate проверяет обязательные поля и ограничения card payload.
func (payload *CardPayload) Validate() error {
	if payload == nil {
		return ErrInvalidCardPayload
	}

	if !utf8.ValidString(payload.Number) ||
		!utf8.ValidString(payload.Cardholder) ||
		!utf8.ValidString(payload.CVV) {
		return ErrInvalidCardPayload
	}
	if strings.TrimSpace(payload.Number) == "" {
		return ErrInvalidCardPayload
	}

	if (payload.ExpiryMonth == nil) != (payload.ExpiryYear == nil) {
		return ErrInvalidCardPayload
	}
	if payload.ExpiryMonth != nil && (*payload.ExpiryMonth < 1 || *payload.ExpiryMonth > 12) {
		return ErrInvalidCardPayload
	}

	return validatePayloadMetadata(payload.Metadata, ErrInvalidCardPayload)
}

// RecordType возвращает тип card-записи.
func (*CardPayload) RecordType() RecordType {
	return RecordTypeCard
}

func (*CardPayload) recordPayload() {
	// запрещает реализацию RecordPayload вне пакета model
}

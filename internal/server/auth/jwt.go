package auth

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	jwtIssuer   = "gopherkeeper"
	jwtAudience = "gopherkeeper-api"
)

// ErrInvalidToken означает, что bearer token отсутствует, повреждён,
// просрочен или не подходит для GophKeeper API.
var ErrInvalidToken = errors.New("invalid token")

type clock interface {
	Now() time.Time
}

type systemClock struct{}

func (systemClock) Now() time.Time {
	return time.Now()
}

type jwtClaims struct {
	jwt.RegisteredClaims
}

// JWTTokenManager выпускает и проверяет JWT access token'ы.
type JWTTokenManager struct {
	secret []byte
	ttl    time.Duration
	clock  clock
}

// NewJWTTokenManager создаёт JWTTokenManager с переданным секретом и TTL.
func NewJWTTokenManager(secret []byte, ttl time.Duration) *JWTTokenManager {
	return newJWTTokenManager(secret, ttl, systemClock{})
}

func newJWTTokenManager(secret []byte, ttl time.Duration, clock clock) *JWTTokenManager {
	return &JWTTokenManager{
		secret: append([]byte(nil), secret...),
		ttl:    ttl,
		clock:  clock,
	}
}

// Issue выпускает подписанный bearer token для пользователя и возвращает
// момент окончания срока его действия.
func (m *JWTTokenManager) Issue(ctx context.Context, userID int64) (string, time.Time, error) {
	if err := ctx.Err(); err != nil {
		return "", time.Time{}, fmt.Errorf("issue JWT token: %w", err)
	}

	if userID <= 0 {
		return "", time.Time{}, errors.New("user ID must be positive")
	}

	now := m.clock.Now().UTC()
	expiresAt := now.Add(m.ttl)
	claims := jwtClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatInt(userID, 10),
			Issuer:    jwtIssuer,
			Audience:  jwt.ClaimStrings{jwtAudience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &claims)
	tokenString, err := token.SignedString(m.secret)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("sign JWT token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// Validate проверяет bearer token и возвращает идентификатор пользователя из
// subject claim.
func (m *JWTTokenManager) Validate(ctx context.Context, tokenString string) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, fmt.Errorf("validate JWT token: %w", err)
	}

	claims := &jwtClaims{}
	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		m.signingKey,
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(jwtIssuer),
		jwt.WithAudience(jwtAudience),
		jwt.WithIssuedAt(),
		jwt.WithTimeFunc(m.clock.Now),
	)
	if err != nil || !token.Valid {
		return 0, ErrInvalidToken
	}

	userID, err := strconv.ParseInt(claims.Subject, 10, 64)
	if err != nil || userID <= 0 {
		return 0, ErrInvalidToken
	}

	return userID, nil
}

func (m *JWTTokenManager) signingKey(_ *jwt.Token) (any, error) {
	return m.secret, nil
}

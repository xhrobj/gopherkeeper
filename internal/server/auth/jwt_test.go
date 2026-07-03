package auth

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type fixedClock struct {
	now time.Time
}

func (c fixedClock) Now() time.Time {
	return c.now
}

var (
	testNow      = time.Date(2026, time.July, 3, 12, 0, 0, 0, time.UTC)
	testSecret   = []byte("0123456789abcdef0123456789abcdef")
	testOtherKey = []byte("abcdef0123456789abcdef0123456789")
	testTokenTTL = 15 * time.Minute
	testUserID   = int64(42)
)

func TestJWTTokenManager_IssueAndValidate(t *testing.T) {
	manager := newTestJWTTokenManager(testSecret)

	token, expiresAt, err := manager.Issue(context.Background(), testUserID)
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}

	if token == "" {
		t.Fatal("Issue() token is empty")
	}

	wantExpiresAt := testNow.Add(testTokenTTL)
	if !expiresAt.Equal(wantExpiresAt) {
		t.Fatalf("Issue() expiresAt = %v, want %v", expiresAt, wantExpiresAt)
	}

	userID, err := manager.Validate(context.Background(), token)
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	if userID != testUserID {
		t.Fatalf("Validate() userID = %d, want %d", userID, testUserID)
	}
}

func TestJWTTokenManager_IssueRejectsInvalidUserID(t *testing.T) {
	manager := newTestJWTTokenManager(testSecret)

	_, _, err := manager.Issue(context.Background(), 0)
	if err == nil {
		t.Fatal("Issue() error = nil, want invalid user ID error")
	}
}

func TestJWTTokenManager_IssueReturnsContextError(t *testing.T) {
	manager := newTestJWTTokenManager(testSecret)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, _, err := manager.Issue(ctx, testUserID)
	if err == nil {
		t.Fatal("Issue() error = nil, want context error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Issue() error = %v, want context.Canceled", err)
	}
}

func TestJWTTokenManager_ValidateRejectsInvalidToken(t *testing.T) {
	manager := newTestJWTTokenManager(testSecret)

	tests := []struct {
		name  string
		token string
	}{
		{
			name:  "empty token",
			token: "",
		},
		{
			name:  "malformed token",
			token: "not-a-jwt",
		},
		{
			name: "expired token",
			token: signTestToken(t, testSecret, jwt.SigningMethodHS256, jwt.RegisteredClaims{
				Subject:   "42",
				Issuer:    jwtIssuer,
				Audience:  jwt.ClaimStrings{jwtAudience},
				IssuedAt:  jwt.NewNumericDate(testNow.Add(-30 * time.Minute)),
				ExpiresAt: jwt.NewNumericDate(testNow.Add(-time.Minute)),
			}),
		},
		{
			name:  "wrong secret",
			token: signValidTestToken(t, testOtherKey),
		},
		{
			name:  "wrong signing method",
			token: signTestToken(t, testSecret, jwt.SigningMethodHS512, validTestClaims("42")),
		},
		{
			name: "wrong issuer",
			token: signTestToken(t, testSecret, jwt.SigningMethodHS256, jwt.RegisteredClaims{
				Subject:   "42",
				Issuer:    "other-service",
				Audience:  jwt.ClaimStrings{jwtAudience},
				IssuedAt:  jwt.NewNumericDate(testNow),
				ExpiresAt: jwt.NewNumericDate(testNow.Add(testTokenTTL)),
			}),
		},
		{
			name: "wrong audience",
			token: signTestToken(t, testSecret, jwt.SigningMethodHS256, jwt.RegisteredClaims{
				Subject:   "42",
				Issuer:    jwtIssuer,
				Audience:  jwt.ClaimStrings{"other-api"},
				IssuedAt:  jwt.NewNumericDate(testNow),
				ExpiresAt: jwt.NewNumericDate(testNow.Add(testTokenTTL)),
			}),
		},
		{
			name: "missing expiration",
			token: signTestToken(t, testSecret, jwt.SigningMethodHS256, jwt.RegisteredClaims{
				Subject:  "42",
				Issuer:   jwtIssuer,
				Audience: jwt.ClaimStrings{jwtAudience},
				IssuedAt: jwt.NewNumericDate(testNow),
			}),
		},
		{
			name:  "invalid subject",
			token: signTestToken(t, testSecret, jwt.SigningMethodHS256, validTestClaims("not-a-number")),
		},
		{
			name:  "zero subject",
			token: signTestToken(t, testSecret, jwt.SigningMethodHS256, validTestClaims("0")),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := manager.Validate(context.Background(), tt.token)
			if !errors.Is(err, ErrInvalidToken) {
				t.Fatalf("Validate() error = %v, want ErrInvalidToken", err)
			}
		})
	}
}

func TestJWTTokenManager_ValidateReturnsContextError(t *testing.T) {
	manager := newTestJWTTokenManager(testSecret)
	token := signValidTestToken(t, testSecret)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := manager.Validate(ctx, token)
	if err == nil {
		t.Fatal("Validate() error = nil, want context error")
	}

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Validate() error = %v, want context.Canceled", err)
	}
}

func newTestJWTTokenManager(secret []byte) *JWTTokenManager {
	return newJWTTokenManager(secret, testTokenTTL, fixedClock{now: testNow})
}

func signValidTestToken(t *testing.T, secret []byte) string {
	t.Helper()

	return signTestToken(t, secret, jwt.SigningMethodHS256, validTestClaims("42"))
}

func validTestClaims(subject string) jwt.RegisteredClaims {
	return jwt.RegisteredClaims{
		Subject:   subject,
		Issuer:    jwtIssuer,
		Audience:  jwt.ClaimStrings{jwtAudience},
		IssuedAt:  jwt.NewNumericDate(testNow),
		ExpiresAt: jwt.NewNumericDate(testNow.Add(testTokenTTL)),
	}
}

func signTestToken(
	t *testing.T,
	secret []byte,
	method jwt.SigningMethod,
	claims jwt.RegisteredClaims,
) string {
	t.Helper()

	token := jwt.NewWithClaims(method, &jwtClaims{RegisteredClaims: claims})
	tokenString, err := token.SignedString(secret)
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	return tokenString
}

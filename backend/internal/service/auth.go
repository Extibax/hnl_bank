package service

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/juanbedoya/hnl-bank/backend/internal/id"
	"github.com/juanbedoya/hnl-bank/backend/internal/model"
	"github.com/juanbedoya/hnl-bank/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrEmailExists indicates a duplicate registration email.
	ErrEmailExists = errors.New("email already exists")
	// ErrInvalidCredentials indicates a bad email/password pair.
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrInvalidToken indicates a missing, malformed, expired or blacklisted token.
	ErrInvalidToken = errors.New("invalid or expired token")
	// ErrUnauthorized indicates the caller lacks permission for the resource.
	ErrUnauthorized = errors.New("unauthorized")
)

const tokenTTL = 24 * time.Hour

// AuthService handles registration, login and token lifecycle.
type AuthService interface {
	Register(ctx context.Context, email, password, fullName string) (*model.User, error)
	Login(ctx context.Context, email, password string) (*model.User, string, error)
	Logout(ctx context.Context, token string) error
	ValidateToken(ctx context.Context, token string) (string, error)
}

type authService struct {
	users     repository.UserRepository
	jwtSecret []byte
	blacklist map[string]time.Time
	mu        sync.RWMutex
}

// NewAuthService builds an AuthService.
func NewAuthService(users repository.UserRepository, jwtSecret string) AuthService {
	return &authService{
		users:     users,
		jwtSecret: []byte(jwtSecret),
		blacklist: make(map[string]time.Time),
	}
}

func (s *authService) Register(ctx context.Context, email, password, fullName string) (*model.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	u := &model.User{
		ID:           id.New(),
		Email:        email,
		PasswordHash: string(hash),
		FullName:     fullName,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
	if err := s.users.Create(ctx, u); err != nil {
		if errors.Is(err, repository.ErrEmailExists) {
			return nil, ErrEmailExists
		}
		return nil, err
	}
	return u, nil
}

func (s *authService) Login(ctx context.Context, email, password string) (*model.User, string, error) {
	u, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, "", ErrInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password)); err != nil {
		return nil, "", ErrInvalidCredentials
	}
	token, err := s.signToken(u.ID)
	if err != nil {
		return nil, "", err
	}
	return u, token, nil
}

func (s *authService) Logout(ctx context.Context, token string) error {
	claims := &jwt.RegisteredClaims{}
	_, _, err := jwt.NewParser().ParseUnverified(token, claims)
	if err != nil {
		return ErrInvalidToken
	}
	s.mu.Lock()
	s.blacklist[token] = claims.ExpiresAt.Time
	s.mu.Unlock()
	return nil
}

func (s *authService) ValidateToken(ctx context.Context, token string) (string, error) {
	s.mu.RLock()
	_, blacklisted := s.blacklist[token]
	s.mu.RUnlock()
	if blacklisted {
		return "", ErrInvalidToken
	}
	claims := &jwt.RegisteredClaims{}
	parsed, err := jwt.ParseWithClaims(token, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return s.jwtSecret, nil
	})
	if err != nil || !parsed.Valid {
		return "", ErrInvalidToken
	}
	return claims.Subject, nil
}

func (s *authService) signToken(userID string) (string, error) {
	now := time.Now().UTC()
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}
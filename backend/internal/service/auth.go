package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
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
	accounts  repository.AccountRepository
	jwtSecret []byte
	blacklist map[string]time.Time
	mu        sync.RWMutex
}

// NewAuthService builds an AuthService.
func NewAuthService(users repository.UserRepository, accounts repository.AccountRepository, jwtSecret string) AuthService {
	return &authService{
		users:     users,
		accounts:  accounts,
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
	if err := s.createDefaultAccount(ctx, u.ID); err != nil {
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

// createDefaultAccount opens a savings account for a newly registered user.
func (s *authService) createDefaultAccount(ctx context.Context, userID string) error {
	number, err := s.uniqueAccountNumber(ctx)
	if err != nil {
		return err
	}
	acct := &model.Account{
		ID:            id.New(),
		UserID:        userID,
		AccountNumber: number,
		TigerBeetleID: repository.NewAccountID(),
		AccountType:   "savings",
		Currency:      "USD",
		CreatedAt:     time.Now().UTC(),
	}
	return s.accounts.Create(ctx, acct)
}

// uniqueAccountNumber generates a free account number in the seed format.
func (s *authService) uniqueAccountNumber(ctx context.Context) (string, error) {
	for i := 0; i < 16; i++ {
		candidate := randomAccountNumber()
		if _, err := s.accounts.FindByNumber(ctx, candidate); err != nil {
			if errors.Is(err, repository.ErrAccountNotFound) {
				return candidate, nil
			}
			return "", err
		}
	}
	return "", errors.New("could not allocate an account number")
}

func randomAccountNumber() string {
	var b [12]byte
	_, _ = rand.Read(b[:])
	grp := func(x, y byte) int { return (int(x)<<8 | int(y)) % 10000 }
	return fmt.Sprintf("4001-%04d-%04d-%04d", grp(b[0], b[1]), grp(b[2], b[3]), grp(b[4], b[5]))
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
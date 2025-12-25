package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserNotFound       = errors.New("user not found")
	ErrUserExists         = errors.New("user already exists")
)

// User represents a user account
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never expose password hash
	Role         string    `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// UserStore manages user accounts
type UserStore interface {
	GetUser(username string) (*User, error)
	CreateUser(username, password, role string) (*User, error)
	UpdatePassword(username, newPassword string) error
	ValidateCredentials(username, password string) (*User, error)
	ListUsers() ([]*User, error)
	DeleteUser(username string) error
}

// InMemoryUserStore is a simple in-memory user store (for development)
type InMemoryUserStore struct {
	users map[string]*User
	mu    sync.RWMutex
}

// NewInMemoryUserStore creates a new in-memory user store
func NewInMemoryUserStore() *InMemoryUserStore {
	return &InMemoryUserStore{
		users: make(map[string]*User),
	}
}

// GetUser retrieves a user by username
func (s *InMemoryUserStore) GetUser(username string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[username]
	if !exists {
		return nil, ErrUserNotFound
	}
	return user, nil
}

// CreateUser creates a new user with hashed password
func (s *InMemoryUserStore) CreateUser(username, password, role string) (*User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; exists {
		return nil, ErrUserExists
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &User{
		ID:           generateUserID(),
		Username:     username,
		PasswordHash: string(hashedPassword),
		Role:         role,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	s.users[username] = user
	return user, nil
}

// UpdatePassword updates a user's password
func (s *InMemoryUserStore) UpdatePassword(username, newPassword string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[username]
	if !exists {
		return ErrUserNotFound
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hashedPassword)
	user.UpdatedAt = time.Now()
	return nil
}

// ValidateCredentials checks if username and password are valid
func (s *InMemoryUserStore) ValidateCredentials(username, password string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[username]
	if !exists {
		return nil, ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidCredentials
	}

	return user, nil
}

// ListUsers returns all users (excluding password hashes)
func (s *InMemoryUserStore) ListUsers() ([]*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}
	return users, nil
}

// DeleteUser removes a user
func (s *InMemoryUserStore) DeleteUser(username string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[username]; !exists {
		return ErrUserNotFound
	}

	delete(s.users, username)
	return nil
}

// InitializeDefaultUsers creates default users from environment variables
func InitializeDefaultUsers(store UserStore) error {
	// Check for custom users from environment
	customUsers := os.Getenv("AUTH_USERS")
	if customUsers != "" {
		// Format: username:password:role,username2:password2:role2
		users := strings.Split(customUsers, ",")
		for _, userStr := range users {
			parts := strings.Split(userStr, ":")
			if len(parts) != 3 {
				continue
			}
			username, password, role := parts[0], parts[1], parts[2]

			// Check if user already exists
			if _, err := store.GetUser(username); err == ErrUserNotFound {
				if _, err := store.CreateUser(username, password, role); err != nil {
					return err
				}
			}
		}
		return nil
	}

	// Create default users if no custom users specified
	defaultUsers := []struct {
		username string
		password string
		role     string
	}{
		{"admin", getEnvOrDefault("ADMIN_PASSWORD", "admin123"), "admin"},
		{"viewer", getEnvOrDefault("VIEWER_PASSWORD", "viewer123"), "viewer"},
		{"operator", getEnvOrDefault("OPERATOR_PASSWORD", "operator123"), "operator"},
	}

	for _, du := range defaultUsers {
		// Only create if doesn't exist
		if _, err := store.GetUser(du.username); err == ErrUserNotFound {
			if _, err := store.CreateUser(du.username, du.password, du.role); err != nil {
				return err
			}
		}
	}

	return nil
}

// generateUserID creates a random user ID
func generateUserID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

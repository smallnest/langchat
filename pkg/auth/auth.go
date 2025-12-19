package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// User represents a user in the system
type User struct {
	ID        string     `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Password  string     `json:"-"` // Never expose password
	Roles     []string   `json:"roles"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	LastLogin *time.Time `json:"last_login,omitempty"`
	Active    bool       `json:"active"`
}

// JWTClaims represents the JWT claims structure (must match middleware)
type JWTClaims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest represents a registration request
type RegisterRequest struct {
	Username string `json:"username" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresIn    int64     `json:"expires_in"`
	User         *UserInfo `json:"user"`
}

// UserInfo represents user information for clients
type UserInfo struct {
	ID       string   `json:"id"`
	Username string   `json:"username"`
	Email    string   `json:"email"`
	Roles    []string `json:"roles"`
}

// AuthService provides authentication services
type AuthService struct {
	users         map[string]*User  // In-memory user store (use database in production)
	refreshTokens map[string]string // Refresh token storage
	secretKey     string
	tokenExpiry   time.Duration
	refreshExpiry time.Duration
}

// NewAuthService creates a new authentication service
func NewAuthService(secretKey string, tokenExpiry, refreshExpiry time.Duration) *AuthService {
	return &AuthService{
		users:         make(map[string]*User),
		refreshTokens: make(map[string]string),
		secretKey:     secretKey,
		tokenExpiry:   tokenExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// CreateUser creates a new user (for testing/demo)
func (a *AuthService) CreateUser(username, email, password string, roles []string) (*User, error) {
	// Check if user already exists
	if _, exists := a.users[username]; exists {
		return nil, fmt.Errorf("user already exists")
	}

	// Hash password (simple hash for demo, use bcrypt in production)
	hashedPassword := a.hashPassword(password)

	user := &User{
		ID:        a.generateID(),
		Username:  username,
		Email:     email,
		Password:  hashedPassword,
		Roles:     roles,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}

	a.users[username] = user
	return user, nil
}

// Login authenticates a user and returns tokens
func (a *AuthService) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	user, exists := a.users[req.Username]
	if !exists {
		return nil, fmt.Errorf("invalid credentials")
	}

	if !user.Active {
		return nil, fmt.Errorf("account is inactive")
	}

	// Verify password
	if !a.verifyPassword(req.Password, user.Password) {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Update last login
	now := time.Now()
	user.LastLogin = &now
	user.UpdatedAt = now

	// Generate tokens
	accessToken, err := a.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := a.generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token
	a.refreshTokens[refreshToken] = user.ID

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(a.tokenExpiry.Seconds()),
		User: &UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Roles:    user.Roles,
		},
	}, nil
}

// Register creates a new user account
func (a *AuthService) Register(ctx context.Context, req *RegisterRequest) (*LoginResponse, error) {
	// Check if user already exists
	if _, exists := a.users[req.Username]; exists {
		return nil, fmt.Errorf("username already exists")
	}

	// Create user
	user, err := a.CreateUser(req.Username, req.Email, req.Password, []string{"user"})
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Generate tokens
	accessToken, err := a.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	refreshToken, err := a.generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token
	a.refreshTokens[refreshToken] = user.ID

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(a.tokenExpiry.Seconds()),
		User: &UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Roles:    user.Roles,
		},
	}, nil
}

// RefreshToken generates a new access token using a refresh token
func (a *AuthService) RefreshToken(ctx context.Context, refreshToken string) (*LoginResponse, error) {
	userID, exists := a.refreshTokens[refreshToken]
	if !exists {
		return nil, fmt.Errorf("invalid refresh token")
	}

	// Find user
	var user *User
	for _, u := range a.users {
		if u.ID == userID {
			user = u
			break
		}
	}

	if user == nil || !user.Active {
		delete(a.refreshTokens, refreshToken)
		return nil, fmt.Errorf("user not found or inactive")
	}

	// Generate new tokens
	accessToken, err := a.generateAccessToken(user)
	if err != nil {
		return nil, fmt.Errorf("failed to generate access token: %w", err)
	}

	newRefreshToken, err := a.generateRefreshToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Update refresh tokens
	delete(a.refreshTokens, refreshToken)
	a.refreshTokens[newRefreshToken] = user.ID

	return &LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: newRefreshToken,
		ExpiresIn:    int64(a.tokenExpiry.Seconds()),
		User: &UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Roles:    user.Roles,
		},
	}, nil
}

// Logout invalidates the refresh token
func (a *AuthService) Logout(ctx context.Context, refreshToken string) error {
	delete(a.refreshTokens, refreshToken)
	return nil
}

// GetUserByID retrieves a user by ID
func (a *AuthService) GetUserByID(userID string) (*User, bool) {
	for _, user := range a.users {
		if user.ID == userID {
			return user, true
		}
	}
	return nil, false
}

// CreateDemoUsers creates demo users for testing
func (a *AuthService) CreateDemoUsers() error {
	// Create admin user
	_, err := a.CreateUser("admin", "admin@example.com", "admin123", []string{"admin", "user"})
	if err != nil {
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	// Create regular user
	_, err = a.CreateUser("user", "user@example.com", "user123", []string{"user"})
	if err != nil {
		return fmt.Errorf("failed to create regular user: %w", err)
	}

	return nil
}

// Helper functions

func (a *AuthService) generateID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return base64.URLEncoding.EncodeToString(b)
}

func (a *AuthService) generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

func (a *AuthService) hashPassword(password string) string {
	// Simple hash for demo - use bcrypt in production
	return base64.StdEncoding.EncodeToString([]byte(password + a.secretKey))
}

func (a *AuthService) verifyPassword(password, hash string) bool {
	hashed := a.hashPassword(password)
	return hashed == hash
}

func (a *AuthService) generateAccessToken(user *User) (string, error) {
	claims := JWTClaims{
		UserID:   user.ID,
		Username: user.Username,
		Roles:    user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "chat-agent",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.secretKey))
}

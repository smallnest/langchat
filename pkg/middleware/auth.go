package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/smallnest/langchat/pkg/auth"
)

// AuthMiddleware provides JWT authentication middleware
type AuthMiddleware struct {
	secretKey     string
	tokenExpiry   time.Duration
	refreshExpiry time.Duration
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(secretKey string, tokenExpiry, refreshExpiry time.Duration) *AuthMiddleware {
	return &AuthMiddleware{
		secretKey:     secretKey,
		tokenExpiry:   tokenExpiry,
		refreshExpiry: refreshExpiry,
	}
}

// GenerateToken generates a new JWT token for the given user
func (a *AuthMiddleware) GenerateToken(userID, username string, roles []string) (string, error) {
	claims := auth.JWTClaims{
		UserID:   userID,
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(a.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "chat-agent",
			Subject:   userID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(a.secretKey))
}

// ValidateToken validates the JWT token and returns the claims
func (a *AuthMiddleware) ValidateToken(tokenString string) (*auth.JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &auth.JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, jwt.ErrSignatureInvalid
		}
		return []byte(a.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*auth.JWTClaims); ok && token.Valid {
		return claims, nil
	}

	return nil, jwt.ErrInvalidKey
}

// Middleware returns an HTTP middleware function for authentication
func (a *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip authentication for health checks and public endpoints
		if a.isPublicEndpoint(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}

		// Extract token from Authorization header
		authHeader := r.Header.Get("Authorization")
		var tokenString string

		if authHeader != "" {
			// Check if the token has the Bearer prefix
			if !strings.HasPrefix(authHeader, "Bearer ") {
				http.Error(w, "Invalid authorization header format", http.StatusUnauthorized)
				return
			}
			tokenString = strings.TrimPrefix(authHeader, "Bearer ")
		} else {
			// Check for token in cookie
			cookie, err := r.Cookie("access_token")
			if err != nil {
				http.Error(w, "Authorization header or cookie required", http.StatusUnauthorized)
				return
			}
			tokenString = cookie.Value
		}

		claims, err := a.ValidateToken(tokenString)
		if err != nil {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Add user information to request context
		ctx := a.setUserContext(r.Context(), claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// isPublicEndpoint checks if the endpoint is public and doesn't require authentication
func (a *AuthMiddleware) isPublicEndpoint(path string) bool {
	publicPaths := []string{
		"/health",
		"/ready",
		"/info",
		"/metrics",
		"/api/config",
		"/login",
		"/register",
	}

	for _, publicPath := range publicPaths {
		if strings.HasPrefix(path, publicPath) {
			return true
		}
	}

	return false
}

// contextKey is the type for context keys
type contextKey string

const userContextKey contextKey = "user"

// setUserContext adds user information to the request context
func (a *AuthMiddleware) setUserContext(ctx context.Context, claims *auth.JWTClaims) context.Context {
	return context.WithValue(ctx, userContextKey, claims)
}

// GetUserFromContext retrieves user information from the request context
func GetUserFromContext(ctx context.Context) (*auth.JWTClaims, bool) {
	user, ok := ctx.Value(userContextKey).(*auth.JWTClaims)
	return user, ok
}

// HasRole checks if the user has the specified role
func (a *AuthMiddleware) HasRole(ctx context.Context, role string) bool {
	user, ok := GetUserFromContext(ctx)
	if !ok {
		return false
	}

	for _, userRole := range user.Roles {
		if userRole == role {
			return true
		}
	}

	return false
}

// RequireRole creates a middleware that requires the user to have the specified role
func (a *AuthMiddleware) RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !a.HasRole(r.Context(), role) {
				http.Error(w, "Insufficient permissions", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

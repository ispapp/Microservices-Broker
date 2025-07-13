package lib

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type AuthMethod int

const (
	AuthMethodJWT AuthMethod = iota
	AuthMethodAPIKey
)

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret   string
	APIKeys     map[string]string // API Key -> Service Name
	TokenExpiry time.Duration
	EnableAuth  bool
	AuthMethod  AuthMethod
}

// AuthManager handles authentication logic
type AuthManager struct {
	config *AuthConfig
}

// JWTClaims represents JWT token claims
type JWTClaims struct {
	ServiceName string `json:"service_name"`
	jwt.RegisteredClaims
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(config *AuthConfig) *AuthManager {
	if config.JWTSecret == "" {
		config.JWTSecret = generateRandomKey(32)
	}
	if config.TokenExpiry == 0 {
		config.TokenExpiry = time.Hour * 24 // 24 hours default
	}
	if config.APIKeys == nil {
		config.APIKeys = make(map[string]string)
	}
	return &AuthManager{config: config}
}

// GenerateAPIKey generates a new API key for a service
func (am *AuthManager) GenerateAPIKey(serviceName string) string {
	apiKey := generateRandomKey(32)
	am.config.APIKeys[apiKey] = serviceName
	return apiKey
}

// GenerateJWT generates a JWT token for a service
func (am *AuthManager) GenerateJWT(serviceName string) (string, error) {
	claims := JWTClaims{
		ServiceName: serviceName,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(am.config.TokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "microservices-broker",
			Subject:   serviceName,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(am.config.JWTSecret))
}

// ValidateJWT validates a JWT token and returns the service name
func (am *AuthManager) ValidateJWT(tokenString string) (string, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(am.config.JWTSecret), nil
	})

	if err != nil {
		return "", err
	}

	if claims, ok := token.Claims.(*JWTClaims); ok && token.Valid {
		return claims.ServiceName, nil
	}

	return "", fmt.Errorf("invalid token")
}

// ValidateAPIKey validates an API key and returns the service name
func (am *AuthManager) ValidateAPIKey(apiKey string) (string, error) {
	if serviceName, exists := am.config.APIKeys[apiKey]; exists {
		return serviceName, nil
	}
	return "", fmt.Errorf("invalid API key")
}

// UnaryInterceptor returns a gRPC unary interceptor for authentication
func (am *AuthManager) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		// Skip authentication if disabled
		if !am.config.EnableAuth {
			return handler(ctx, req)
		}

		// Skip authentication for ping method (health check)
		if strings.HasSuffix(info.FullMethod, "/Ping") {
			return handler(ctx, req)
		}

		serviceName, err := am.authenticate(ctx)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
		}

		// Add service name to context for use in handlers
		ctx = context.WithValue(ctx, serviceNameCtxKey{}, serviceName)
		return handler(ctx, req)
	}
}

// StreamInterceptor returns a gRPC stream interceptor for authentication
func (am *AuthManager) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		// Skip authentication if disabled
		if !am.config.EnableAuth {
			return handler(srv, ss)
		}

		serviceName, err := am.authenticate(ss.Context())
		if err != nil {
			return status.Errorf(codes.Unauthenticated, "authentication failed: %v", err)
		}

		// Create a new context with service name
		ctx := context.WithValue(ss.Context(), serviceNameCtxKey{}, serviceName)
		wrapped := &wrappedStream{ss, ctx}
		return handler(srv, wrapped)
	}
}

// authenticate extracts and validates authentication from context
func (am *AuthManager) authenticate(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", fmt.Errorf("missing metadata")
	}

	switch am.config.AuthMethod {
	case AuthMethodJWT:
		return am.authenticateJWT(md)
	case AuthMethodAPIKey:
		return am.authenticateAPIKey(md)
	default:
		return "", fmt.Errorf("unsupported authentication method")
	}
}

// authenticateJWT validates JWT token from metadata
func (am *AuthManager) authenticateJWT(md metadata.MD) (string, error) {
	values := md.Get("authorization")
	if len(values) == 0 {
		return "", fmt.Errorf("missing authorization header")
	}

	token := values[0]
	if !strings.HasPrefix(token, "Bearer ") {
		return "", fmt.Errorf("invalid authorization format")
	}

	tokenString := strings.TrimPrefix(token, "Bearer ")
	return am.ValidateJWT(tokenString)
}

// authenticateAPIKey validates API key from metadata
func (am *AuthManager) authenticateAPIKey(md metadata.MD) (string, error) {
	values := md.Get("x-api-key")
	if len(values) == 0 {
		return "", fmt.Errorf("missing API key")
	}

	return am.ValidateAPIKey(values[0])
}

// wrappedStream wraps a grpc.ServerStream with a custom context
type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (w *wrappedStream) Context() context.Context {
	return w.ctx
}

// generateRandomKey generates a random key of specified length
func generateRandomKey(length int) string {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a simple hash if random generation fails
		hash := sha256.Sum256([]byte(fmt.Sprintf("%d", time.Now().UnixNano())))
		return hex.EncodeToString(hash[:])[:length]
	}
	return hex.EncodeToString(bytes)
}

// serviceNameCtxKey is a custom type for context keys to avoid collisions
type serviceNameCtxKey struct{}

// GetServiceNameFromContext extracts service name from context
func GetServiceNameFromContext(ctx context.Context) string {
	if serviceName, ok := ctx.Value(serviceNameCtxKey{}).(string); ok {
		return serviceName
	}
	return ""
}

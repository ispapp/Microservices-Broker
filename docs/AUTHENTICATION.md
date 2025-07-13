# Microservices Broker Authentication

This document describes the authentication system implemented for the Microservices Broker.

## Overview

The Microservices Broker now includes a comprehensive authentication system that supports:
- **JWT (JSON Web Tokens)**: Stateless authentication with never-expiring tokens
- **API Keys**: Simple key-based authentication for service-to-service communication
- **TLS/SSL**: Optional transport layer security
- **gRPC Interceptors**: Automatic authentication validation for all requests

## Features

### Authentication Methods

1. **JWT Authentication** (Default)
   - Uses HMAC-SHA256 signing
   - Tokens never expire (stateless, long-lived)
   - Contains service name in claims
   - Secure random JWT secret generation

2. **API Key Authentication**
   - Simple key-value mapping
   - Keys are randomly generated (64 characters)
   - Direct service name mapping

### Security Features

- **gRPC Interceptors**: Automatic authentication for all RPC calls
- **TLS Support**: Optional encrypted transport
- **Configurable**: Enable/disable authentication per deployment
- **Service Isolation**: Each service has its own authentication credentials

## Quick Start

### 1. Initialize Configuration

```bash
# Create default configuration file
./broker config init-config

# Or specify custom path
./broker config init-config --config /path/to/config.json
```

### 2. Generate Authentication Credentials

#### For JWT Authentication (Default)

```bash
# Generate JWT token for a service
./broker auth generate-jwt --service my-service

# Example output:
# Generated JWT token for service 'my-service': eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

#### For API Key Authentication

```bash
# Set authentication method to API key
./broker config set-auth-method --method apikey

# Generate API key for a service
./broker auth generate-key --service my-service

# Example output:
# Generated API key for service 'my-service': a1b2c3d4e5f6...
```

### 3. Start the Server

```bash
# Start with authentication enabled (default)
./broker serve

# Start with custom configuration
./broker serve --config /path/to/config.json

# Start without authentication (NOT recommended for production)
./broker serve --disable-auth
```

### 4. Enable TLS (Optional)

```bash
# Generate TLS certificates (example script provided)
./scripts/generate_tls_certs.sh

# Enable TLS in configuration
./broker config enable-tls --cert server.crt --key server.key
```

## Configuration

### Configuration File Structure

```json
{
  "server": {
    "host": "0.0.0.0",
    "port": "50011",
    "tls_enabled": false,
    "tls_cert_file": "server.crt",
    "tls_key_file": "server.key",
    "tick_seconds": 60,
    "max_stored": 100,
    "max_age": "24h0m0s"
  },
  "auth": {
    "JWTSecret": "randomly-generated-secret",
    "APIKeys": {
      "api-key-1": "service-1",
      "api-key-2": "service-2"
    },
    "EnableAuth": true,
    "AuthMethod": 0
  },
  "database": {
    "path": "broker.db"
  }
}
```

### Authentication Methods

- `AuthMethod: 0` = JWT Authentication
- `AuthMethod: 1` = API Key Authentication

## Client Implementation

### JWT Authentication

```go
// Create authenticated client
client, err := NewAuthenticatedClient("localhost:50011", "my-service", "jwt", false, "")
if err != nil {
    log.Fatal(err)
}

// Set JWT token
client.SetJWTToken("your-jwt-token-here")

// Use client normally
status, err := client.Ping(context.Background())
```

### API Key Authentication

```go
// Create authenticated client
client, err := NewAuthenticatedClient("localhost:50011", "my-service", "apikey", false, "")
if err != nil {
    log.Fatal(err)
}

// Set API key
client.SetAPIKey("your-api-key-here")

// Use client normally
status, err := client.Ping(context.Background())
```

### Headers

#### JWT Authentication
- Header: `authorization`
- Value: `Bearer <jwt-token>`

#### API Key Authentication
- Header: `x-api-key`
- Value: `<api-key>`

## Management Commands

### Authentication Management

```bash
# Generate JWT token
./broker auth generate-jwt --service <service-name>

# Generate API key
./broker auth generate-key --service <service-name>

# List all API keys
./broker auth list-keys

# Remove API key
./broker auth remove-key --key <api-key>
```

### Configuration Management

```bash
# Show current configuration
./broker config show

# Set authentication method
./broker config set-auth-method --method [jwt|apikey]

# Enable TLS
./broker config enable-tls --cert <cert-file> --key <key-file>

# Initialize default configuration
./broker config init-config
```

## Security Considerations

### Production Recommendations

1. **Always Enable Authentication**: Never run in production with `--disable-auth`
2. **Use TLS**: Enable TLS for encrypted transport
3. **Secure Configuration**: Protect configuration files (contains secrets)
4. **Regular Key Rotation**: Periodically rotate API keys and JWT secrets
5. **Network Security**: Use firewalls and network segmentation

### JWT Security

- **Never-Expiring Tokens**: Tokens don't expire, so secure storage is crucial
- **Secret Protection**: JWT secret must be kept secure
- **Token Distribution**: Use secure channels to distribute tokens to services

### API Key Security

- **Unique Keys**: Each service should have its own API key
- **Key Storage**: Store keys securely in environment variables or secret management
- **Key Rotation**: Implement regular key rotation procedures

## Error Handling

### Common Authentication Errors

1. **Missing Authentication**: `missing metadata` or `missing authorization header`
2. **Invalid Token**: `invalid token` or `invalid API key`
3. **Wrong Format**: `invalid authorization format`
4. **Service Mismatch**: Token/key doesn't match expected service

### Error Response Format

All authentication errors return gRPC status code `Unauthenticated` with descriptive messages.

## Examples

Complete client examples are available in the `examples/` directory:

- `examples/auth_client.go` - Demonstrates both JWT and API key authentication

## Migration Guide

### From Unauthenticated to Authenticated

1. **Generate Configuration**:
   ```bash
   ./broker config init-config
   ```

2. **Generate Credentials** for each service:
   ```bash
   ./broker auth generate-jwt --service service-1
   ./broker auth generate-jwt --service service-2
   ```

3. **Update Clients** to include authentication headers

4. **Restart Server** with authentication enabled

### Switching Authentication Methods

1. **Change Method**:
   ```bash
   ./broker config set-auth-method --method apikey
   ```

2. **Generate New Credentials**:
   ```bash
   ./broker auth generate-key --service service-1
   ```

3. **Update Clients** to use new authentication method

4. **Restart Server** to apply changes

## Troubleshooting

### Common Issues

1. **Build Errors**: Ensure all dependencies are installed with `go mod tidy`
2. **Token Validation**: Verify JWT secret matches between client and server
3. **TLS Issues**: Ensure certificate files exist and are readable
4. **Permission Errors**: Check file permissions for configuration and database files

### Debug Mode

Enable verbose logging by setting environment variable:
```bash
export GRPC_GO_LOG_VERBOSITY_LEVEL=99
export GRPC_GO_LOG_SEVERITY_LEVEL=info
```

## API Reference

### Authentication Manager

- `NewAuthManager(config *AuthConfig) *AuthManager`
- `GenerateJWT(serviceName string) (string, error)`
- `GenerateAPIKey(serviceName string) string`
- `ValidateJWT(tokenString string) (string, error)`
- `ValidateAPIKey(apiKey string) (string, error)`

### Configuration

- `LoadConfig(configPath string) (*Config, error)`
- `SaveConfig(configPath string) error`
- `GenerateDefaultConfig(configPath string) error`

### Client

- `NewAuthenticatedClient(address, serviceName, authMethod string, useTLS bool, certFile string) (*AuthenticatedClient, error)`
- `SetJWTToken(token string)`
- `SetAPIKey(apiKey string)`

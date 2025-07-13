package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"testing"
	"time"

	"github.com/ispapp/Microservices-Broker/base/pb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
)

// AuthenticatedClient demonstrates how to use the broker with authentication
type AuthenticatedClient struct {
	conn        *grpc.ClientConn
	client      pb.BrokerClient
	serviceName string
	apiKey      string
	jwtToken    string
	authMethod  string // "jwt" or "apikey"
}

// NewAuthenticatedClient creates a new authenticated client
func NewAuthenticatedClient(address, serviceName, authMethod string, useTLS bool, certFile string) (*AuthenticatedClient, error) {
	var opts []grpc.DialOption

	if useTLS {
		if certFile != "" {
			// Use custom certificate
			creds, err := credentials.NewClientTLSFromFile(certFile, "")
			if err != nil {
				return nil, fmt.Errorf("failed to load TLS credentials: %w", err)
			}
			opts = append(opts, grpc.WithTransportCredentials(creds))
		} else {
			// Use system certificates
			opts = append(opts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
		}
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return &AuthenticatedClient{
		conn:        conn,
		client:      pb.NewBrokerClient(conn),
		serviceName: serviceName,
		authMethod:  authMethod,
	}, nil
}

// SetAPIKey sets the API key for authentication
func (ac *AuthenticatedClient) SetAPIKey(apiKey string) {
	ac.apiKey = apiKey
}

// SetJWTToken sets the JWT token for authentication
func (ac *AuthenticatedClient) SetJWTToken(token string) {
	ac.jwtToken = token
}

// createAuthContext creates a context with authentication metadata
func (ac *AuthenticatedClient) createAuthContext(ctx context.Context) context.Context {
	md := metadata.New(nil)

	switch ac.authMethod {
	case "jwt":
		if ac.jwtToken != "" {
			md.Set("authorization", "Bearer "+ac.jwtToken)
		}
	case "apikey":
		if ac.apiKey != "" {
			md.Set("x-api-key", ac.apiKey)
		}
	}

	return metadata.NewOutgoingContext(ctx, md)
}

// Ping sends a ping request to the broker
func (ac *AuthenticatedClient) Ping(ctx context.Context) (*pb.Status, error) {
	authCtx := ac.createAuthContext(ctx)
	return ac.client.Ping(authCtx, &pb.Identity{From: ac.serviceName})
}

// Send sends a message through the broker
func (ac *AuthenticatedClient) Send(ctx context.Context, to string, data []byte, msgType pb.Type, queue bool) (*pb.Status, error) {
	authCtx := ac.createAuthContext(ctx)

	msg := &pb.Message{
		Data:  data,
		Type:  msgType,
		From:  ac.serviceName,
		To:    to,
		Queue: queue,
	}

	return ac.client.Send(authCtx, msg)
}

// Receive starts receiving messages from the broker
func (ac *AuthenticatedClient) Receive(ctx context.Context) (pb.Broker_ReceiveClient, error) {
	authCtx := ac.createAuthContext(ctx)
	return ac.client.Receive(authCtx, &pb.Identity{From: ac.serviceName})
}

// Cleanup cleans up messages for the service
func (ac *AuthenticatedClient) Cleanup(ctx context.Context) (*pb.Status, error) {
	authCtx := ac.createAuthContext(ctx)
	return ac.client.Cleanup(authCtx, &pb.Identity{From: ac.serviceName})
}

// Close closes the connection
func (ac *AuthenticatedClient) Close() error {
	return ac.conn.Close()
}

// Example usage
func TestAuthentication(t *testing.T) {
	// Example 1: Using API Key authentication
	fmt.Println("=== API Key Authentication Example ===")
	apiKeyClient, err := NewAuthenticatedClient("localhost:50011", "service-1", "apikey", false, "")
	if err != nil {
		t.Fatalf("Failed to create API key client: %v", err)
	}
	defer apiKeyClient.Close()

	// Set API key (you would get this from your configuration)
	apiKeyClient.SetAPIKey("your-api-key-here")

	// Test ping
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	status, err := apiKeyClient.Ping(ctx)
	if err != nil {
		t.Logf("API Key Ping failed: %v", err)
	} else {
		fmt.Printf("API Key Ping response: %s (Success: %t)\n", status.Message, status.Success)
	}

	// Example 2: Using JWT authentication
	fmt.Println("\n=== JWT Authentication Example ===")
	jwtClient, err := NewAuthenticatedClient("localhost:50011", "service-2", "jwt", false, "")
	if err != nil {
		t.Fatalf("Failed to create JWT client: %v", err)
	}
	defer jwtClient.Close()

	// Set JWT token (you would get this from your authentication service)
	jwtClient.SetJWTToken("your-jwt-token-here")

	// Test ping
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	status2, err := jwtClient.Ping(ctx2)
	if err != nil {
		t.Logf("JWT Ping failed: %v", err)
	} else {
		fmt.Printf("JWT Ping response: %s (Success: %t)\n", status2.Message, status2.Success)
	}

	// Example 3: Send and receive messages
	fmt.Println("\n=== Message Exchange Example ===")

	// Send a message
	sendCtx, sendCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer sendCancel()

	sendStatus, err := apiKeyClient.Send(sendCtx, "service-2", []byte("Hello from service-1"), pb.Type_TEXT, true)
	if err != nil {
		t.Logf("Send failed: %v", err)
	} else {
		fmt.Printf("Send response: %s (Success: %t)\n", sendStatus.Message, sendStatus.Success)
	}

	// Start receiving messages
	recvCtx, recvCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer recvCancel()

	stream, err := jwtClient.Receive(recvCtx)
	if err != nil {
		t.Logf("Receive failed: %v", err)
		return
	}

	// Listen for messages
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				t.Logf("Receive stream error: %v", err)
				return
			}
			fmt.Printf("Received message: %s (from: %s, type: %s)\n", string(msg.Data), msg.From, msg.Type)
		}
	}()

	// Wait for messages
	time.Sleep(2 * time.Second)

	fmt.Println("\nExample completed!")
}

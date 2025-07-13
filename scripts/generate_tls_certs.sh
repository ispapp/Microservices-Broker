#!/bin/bash

# Generate TLS certificates for the gRPC server

# Create a directory for certificates
mkdir -p certs
cd certs

# Generate private key
openssl genrsa -out server.key 4096

# Generate certificate signing request
openssl req -new -key server.key -out server.csr -subj "/C=US/ST=State/L=City/O=Organization/OU=OrgUnit/CN=localhost"

# Generate self-signed certificate
openssl x509 -req -days 365 -in server.csr -signkey server.key -out server.crt

# Clean up
rm server.csr

echo "TLS certificates generated:"
echo "  Certificate: certs/server.crt"
echo "  Private Key: certs/server.key"
echo ""
echo "To use with the broker:"
echo "  1. Run: ./broker config enable-tls --cert certs/server.crt --key certs/server.key"
echo "  2. Start server: ./broker serve"
echo ""
echo "Note: This is a self-signed certificate for development only."
echo "For production, use certificates from a trusted CA."

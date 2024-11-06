package grpc

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// SetupGRPCConnection establishes a GRPC connection, optionally using system's TLS certificates
func SetupGRPCConnection(address string, useTLS bool) (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	if useTLS {
		// Load the system's certificate pool
		systemCertPool, err := x509.SystemCertPool()
		if err != nil {
			return nil, fmt.Errorf("failed to load system certificates: %w", err)
		}

		// Create TLS credentials using the system's certificate pool
		creds := credentials.NewTLS(&tls.Config{
			RootCAs:    systemCertPool,
			MinVersion: tls.VersionTLS12,
		})

		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		// Use insecure credentials if TLS is not required
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Dial the GRPC server with the appropriate credentials
	conn, err := grpc.NewClient(address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to gRPC server at %s: %w", address, err)
	}

	return conn, nil
}

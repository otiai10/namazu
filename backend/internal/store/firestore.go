package store

import (
	"context"
	"fmt"
	"log"
	"os"

	"cloud.google.com/go/firestore"
	"google.golang.org/api/option"
)

// FirestoreClient wraps the Firestore client for data persistence
type FirestoreClient struct {
	client    *firestore.Client
	projectID string
	database  string
}

// Ensure FirestoreClient implements Store interface
var _ Store = (*FirestoreClient)(nil)

// FirestoreConfig holds configuration for Firestore client
type FirestoreConfig struct {
	ProjectID   string // GCP Project ID (required)
	Database    string // Database name (optional, defaults to "(default)")
	Credentials string // Path to service account JSON file (optional)
}

// NewFirestoreClient creates a new Firestore client.
// If FIRESTORE_EMULATOR_HOST is set, the client will connect to the emulator.
func NewFirestoreClient(ctx context.Context, cfg FirestoreConfig) (*FirestoreClient, error) {
	if cfg.ProjectID == "" {
		return nil, fmt.Errorf("projectID is required")
	}

	// Check if using emulator
	emulatorHost := os.Getenv("FIRESTORE_EMULATOR_HOST")
	if emulatorHost != "" {
		log.Printf("🔧 Using Firestore Emulator at %s", emulatorHost)
	}

	// Build client options
	var opts []option.ClientOption
	if cfg.Credentials != "" && emulatorHost == "" {
		// Only use credentials file when not using emulator
		opts = append(opts, option.WithCredentialsFile(cfg.Credentials))
	}

	// Use named database if specified
	database := cfg.Database
	if database == "" {
		database = "(default)"
	}

	client, err := firestore.NewClientWithDatabase(ctx, cfg.ProjectID, database, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create firestore client: %w", err)
	}

	return &FirestoreClient{
		client:    client,
		projectID: cfg.ProjectID,
		database:  database,
	}, nil
}

// Close releases resources held by the Firestore client
func (f *FirestoreClient) Close() error {
	if f.client == nil {
		return nil
	}
	return f.client.Close()
}

// Client returns the underlying Firestore client
// This allows access to Firestore operations for higher-level code
func (f *FirestoreClient) Client() *firestore.Client {
	return f.client
}

// ProjectID returns the GCP project ID
func (f *FirestoreClient) ProjectID() string {
	return f.projectID
}

// Database returns the Firestore database name
func (f *FirestoreClient) Database() string {
	return f.database
}

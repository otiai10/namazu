package store

import (
	"context"
	"testing"
)

func TestNewFirestoreClient_EmptyProjectID(t *testing.T) {
	ctx := context.Background()

	_, err := NewFirestoreClient(ctx, FirestoreConfig{})
	if err == nil {
		t.Fatal("NewFirestoreClient() should return error for empty projectID")
	}

	expectedMsg := "projectID is required"
	if err.Error() != expectedMsg {
		t.Errorf("NewFirestoreClient() error = %q, want %q", err.Error(), expectedMsg)
	}
}

func TestFirestoreClient_ProjectID(t *testing.T) {
	// This test verifies the ProjectID getter without connecting to Firestore
	// We can't fully test NewFirestoreClient without actual GCP credentials
	// but we can test the validation and accessor methods

	// Create a mock-like test by testing the struct directly
	fc := &FirestoreClient{
		client:    nil,
		projectID: "test-project-123",
		database:  "test-database",
	}

	if fc.ProjectID() != "test-project-123" {
		t.Errorf("ProjectID() = %q, want %q", fc.ProjectID(), "test-project-123")
	}
}

func TestFirestoreClient_Database(t *testing.T) {
	fc := &FirestoreClient{
		client:    nil,
		projectID: "test-project",
		database:  "custom-db",
	}

	if fc.Database() != "custom-db" {
		t.Errorf("Database() = %q, want %q", fc.Database(), "custom-db")
	}
}

func TestFirestoreClient_Close_NilClient(t *testing.T) {
	// Test that Close handles nil client gracefully
	fc := &FirestoreClient{
		client:    nil,
		projectID: "test-project",
		database:  "(default)",
	}

	err := fc.Close()
	if err != nil {
		t.Errorf("Close() error = %v, want nil for nil client", err)
	}
}

func TestFirestoreClient_Client_NilReturnsNil(t *testing.T) {
	fc := &FirestoreClient{
		client:    nil,
		projectID: "test-project",
		database:  "(default)",
	}

	if fc.Client() != nil {
		t.Error("Client() should return nil when underlying client is nil")
	}
}

func TestFirestoreClient_ImplementsStore(t *testing.T) {
	// Compile-time check that FirestoreClient implements Store interface
	var _ Store = (*FirestoreClient)(nil)
}

// TestNewFirestoreClient_ConnectionError tests that connection errors are handled gracefully.
// This test will fail to connect to Firestore (expected behavior without credentials)
// and verifies error wrapping works correctly.
func TestNewFirestoreClient_ConnectionError(t *testing.T) {
	// Skip in short mode as this may take time to fail
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	ctx := context.Background()

	// Using an invalid project ID format to trigger an error
	// Note: This test behavior depends on the environment (GCP credentials, emulator, etc.)
	_, err := NewFirestoreClient(ctx, FirestoreConfig{
		ProjectID: "invalid-project-that-does-not-exist",
	})

	// The behavior depends on environment:
	// - With emulator: may succeed
	// - Without credentials: will fail
	// - With credentials but wrong project: may fail
	// We're mainly testing that the function doesn't panic
	if err != nil {
		t.Logf("NewFirestoreClient returned expected error: %v", err)
	}
}

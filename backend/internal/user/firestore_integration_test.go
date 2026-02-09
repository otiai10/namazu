//go:build integration

package user

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/otiai10/namazu/backend/internal/store"
)

// TestFirestoreRepository_Integration tests the FirestoreRepository against a real Firestore instance.
// Run with: go test -tags=integration ./internal/user/... -v
//
// Requires environment variables:
//   - NAMAZU_STORE_PROJECT_ID
//   - NAMAZU_STORE_DATABASE (optional, defaults to "(default)")
//   - NAMAZU_STORE_CREDENTIALS (path to service account JSON for local dev)
func TestFirestoreRepository_Integration(t *testing.T) {
	projectID := os.Getenv("NAMAZU_STORE_PROJECT_ID")
	if projectID == "" {
		t.Skip("NAMAZU_STORE_PROJECT_ID not set, skipping integration test")
	}

	ctx := context.Background()

	// Create Firestore client
	client, err := store.NewFirestoreClient(ctx, store.FirestoreConfig{
		ProjectID:   projectID,
		Database:    os.Getenv("NAMAZU_STORE_DATABASE"),
		Credentials: os.Getenv("NAMAZU_STORE_CREDENTIALS"),
	})
	if err != nil {
		t.Fatalf("failed to create Firestore client: %v", err)
	}
	defer client.Close()

	repo := NewFirestoreRepository(client.Client())

	// Generate unique test ID to avoid conflicts
	testUID := "integration-test-" + time.Now().Format("20060102-150405")
	var createdID string

	t.Run("Create user", func(t *testing.T) {
		now := time.Now().UTC()
		user := User{
			UID:         testUID,
			Email:       "test@example.com",
			DisplayName: "Integration Test User",
			Plan:        PlanFree,
			Providers: []LinkedProvider{
				{
					ProviderID:  ProviderGoogle,
					Subject:     "google-" + testUID,
					Email:       "test@gmail.com",
					DisplayName: "Test Google",
					LinkedAt:    now,
				},
			},
			CreatedAt:   now,
			UpdatedAt:   now,
			LastLoginAt: now,
		}

		id, err := repo.Create(ctx, user)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if id == "" {
			t.Fatal("Create returned empty ID")
		}
		createdID = id
		t.Logf("Created user with ID: %s", id)
	})

	t.Run("Get by UID", func(t *testing.T) {
		user, err := repo.GetByUID(ctx, testUID)
		if err != nil {
			t.Fatalf("GetByUID failed: %v", err)
		}
		if user == nil {
			t.Fatal("GetByUID returned nil")
		}
		if user.UID != testUID {
			t.Errorf("UID mismatch: got %s, want %s", user.UID, testUID)
		}
		if user.Email != "test@example.com" {
			t.Errorf("Email mismatch: got %s", user.Email)
		}
		if len(user.Providers) != 1 {
			t.Errorf("Providers count: got %d, want 1", len(user.Providers))
		}
	})

	t.Run("Get by ID", func(t *testing.T) {
		user, err := repo.Get(ctx, createdID)
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if user == nil {
			t.Fatal("Get returned nil")
		}
		if user.ID != createdID {
			t.Errorf("ID mismatch: got %s, want %s", user.ID, createdID)
		}
	})

	t.Run("Duplicate UID returns error", func(t *testing.T) {
		duplicateUser := User{
			UID:       testUID,
			Email:     "duplicate@example.com",
			Plan:      PlanFree,
			CreatedAt: time.Now().UTC(),
		}
		_, err := repo.Create(ctx, duplicateUser)
		if err != ErrDuplicateUID {
			t.Errorf("expected ErrDuplicateUID, got: %v", err)
		}
	})

	t.Run("Update user", func(t *testing.T) {
		user, _ := repo.Get(ctx, createdID)
		user.DisplayName = "Updated Name"
		user.Plan = PlanPro

		err := repo.Update(ctx, createdID, *user)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		updated, _ := repo.Get(ctx, createdID)
		if updated.DisplayName != "Updated Name" {
			t.Errorf("DisplayName not updated: got %s", updated.DisplayName)
		}
		if updated.Plan != PlanPro {
			t.Errorf("Plan not updated: got %s", updated.Plan)
		}
	})

	t.Run("Update last login", func(t *testing.T) {
		newTime := time.Now().UTC().Add(time.Hour)
		err := repo.UpdateLastLogin(ctx, createdID, newTime)
		if err != nil {
			t.Fatalf("UpdateLastLogin failed: %v", err)
		}

		user, _ := repo.Get(ctx, createdID)
		if user.LastLoginAt.Before(newTime.Add(-time.Second)) {
			t.Errorf("LastLoginAt not updated correctly")
		}
	})

	t.Run("Add provider", func(t *testing.T) {
		newProvider := LinkedProvider{
			ProviderID:  ProviderApple,
			Subject:     "apple-" + testUID,
			Email:       "test@icloud.com",
			DisplayName: "Test Apple",
			LinkedAt:    time.Now().UTC(),
		}

		err := repo.AddProvider(ctx, createdID, newProvider)
		if err != nil {
			t.Fatalf("AddProvider failed: %v", err)
		}

		user, _ := repo.Get(ctx, createdID)
		if len(user.Providers) != 2 {
			t.Errorf("Providers count: got %d, want 2", len(user.Providers))
		}
	})

	t.Run("Add duplicate provider returns error", func(t *testing.T) {
		duplicateProvider := LinkedProvider{
			ProviderID: ProviderGoogle,
			Subject:    "another-subject",
			LinkedAt:   time.Now().UTC(),
		}

		err := repo.AddProvider(ctx, createdID, duplicateProvider)
		if err != ErrProviderExists {
			t.Errorf("expected ErrProviderExists, got: %v", err)
		}
	})

	t.Run("Remove provider", func(t *testing.T) {
		err := repo.RemoveProvider(ctx, createdID, ProviderApple)
		if err != nil {
			t.Fatalf("RemoveProvider failed: %v", err)
		}

		user, _ := repo.Get(ctx, createdID)
		if len(user.Providers) != 1 {
			t.Errorf("Providers count: got %d, want 1", len(user.Providers))
		}
	})

	t.Run("Remove non-existent provider returns error", func(t *testing.T) {
		err := repo.RemoveProvider(ctx, createdID, "nonexistent")
		if err != ErrProviderNotFound {
			t.Errorf("expected ErrProviderNotFound, got: %v", err)
		}
	})

	// Cleanup: Delete the test user document
	t.Run("Cleanup", func(t *testing.T) {
		_, err := client.Client().Collection("users").Doc(createdID).Delete(ctx)
		if err != nil {
			t.Logf("Warning: failed to cleanup test user: %v", err)
		} else {
			t.Logf("Cleaned up test user: %s", createdID)
		}
	})
}

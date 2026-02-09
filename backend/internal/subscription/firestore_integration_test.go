//go:build integration

package subscription

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/otiai10/namazu/backend/internal/store"
)

// TestFirestoreRepository_Integration tests the FirestoreRepository against a real Firestore instance.
// Run with: go test -tags=integration ./internal/subscription/... -v
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

	// Generate unique test ID
	testUserID := "integration-test-user-" + time.Now().Format("20060102-150405")
	var createdIDs []string

	t.Run("Create subscription with UserID", func(t *testing.T) {
		sub := Subscription{
			UserID: testUserID,
			Name:   "Integration Test Sub 1",
			Delivery: DeliveryConfig{
				Type:   "webhook",
				URL:    "https://example.com/webhook1",
				Secret: "secret1",
			},
		}

		id, err := repo.Create(ctx, sub)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		if id == "" {
			t.Fatal("Create returned empty ID")
		}
		createdIDs = append(createdIDs, id)
		t.Logf("Created subscription with ID: %s", id)
	})

	t.Run("Create second subscription for same user", func(t *testing.T) {
		sub := Subscription{
			UserID: testUserID,
			Name:   "Integration Test Sub 2",
			Delivery: DeliveryConfig{
				Type:   "webhook",
				URL:    "https://example.com/webhook2",
				Secret: "secret2",
			},
			Filter: &FilterConfig{
				MinScale:    40,
				Prefectures: []string{"東京都", "神奈川県"},
			},
		}

		id, err := repo.Create(ctx, sub)
		if err != nil {
			t.Fatalf("Create failed: %v", err)
		}
		createdIDs = append(createdIDs, id)
		t.Logf("Created subscription with ID: %s", id)
	})

	t.Run("ListByUserID returns user's subscriptions", func(t *testing.T) {
		subs, err := repo.ListByUserID(ctx, testUserID)
		if err != nil {
			t.Fatalf("ListByUserID failed: %v", err)
		}
		if len(subs) != 2 {
			t.Errorf("expected 2 subscriptions, got %d", len(subs))
		}
		for _, sub := range subs {
			if sub.UserID != testUserID {
				t.Errorf("subscription has wrong UserID: %s", sub.UserID)
			}
		}
	})

	t.Run("ListByUserID returns empty for non-existent user", func(t *testing.T) {
		subs, err := repo.ListByUserID(ctx, "non-existent-user")
		if err != nil {
			t.Fatalf("ListByUserID failed: %v", err)
		}
		if len(subs) != 0 {
			t.Errorf("expected 0 subscriptions, got %d", len(subs))
		}
	})

	t.Run("Get subscription preserves UserID", func(t *testing.T) {
		sub, err := repo.Get(ctx, createdIDs[0])
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if sub.UserID != testUserID {
			t.Errorf("UserID mismatch: got %s, want %s", sub.UserID, testUserID)
		}
	})

	t.Run("Update subscription preserves UserID", func(t *testing.T) {
		sub, _ := repo.Get(ctx, createdIDs[0])
		sub.Name = "Updated Name"

		err := repo.Update(ctx, createdIDs[0], *sub)
		if err != nil {
			t.Fatalf("Update failed: %v", err)
		}

		updated, _ := repo.Get(ctx, createdIDs[0])
		if updated.UserID != testUserID {
			t.Errorf("UserID changed after update: got %s", updated.UserID)
		}
		if updated.Name != "Updated Name" {
			t.Errorf("Name not updated: got %s", updated.Name)
		}
	})

	// Cleanup
	t.Run("Cleanup", func(t *testing.T) {
		for _, id := range createdIDs {
			err := repo.Delete(ctx, id)
			if err != nil {
				t.Logf("Warning: failed to cleanup subscription %s: %v", id, err)
			} else {
				t.Logf("Cleaned up subscription: %s", id)
			}
		}
	})
}

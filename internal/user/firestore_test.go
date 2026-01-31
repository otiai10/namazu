package user

import (
	"testing"
	"time"
)

func TestNewFirestoreRepository(t *testing.T) {
	t.Run("creates repository with nil client", func(t *testing.T) {
		repo := NewFirestoreRepository(nil)

		if repo == nil {
			t.Fatal("NewFirestoreRepository returned nil")
		}
		if repo.client != nil {
			t.Error("Expected client to be nil")
		}
	})
}

func TestFirestoreRepository_ImplementsRepository(t *testing.T) {
	// Compile-time check that FirestoreRepository implements Repository interface
	var _ Repository = (*FirestoreRepository)(nil)
}

func TestUserToMap(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("converts user with all fields", func(t *testing.T) {
		user := User{
			ID:          "doc-id",
			UID:         "uid-123",
			Email:       "test@example.com",
			DisplayName: "Test User",
			PictureURL:  "https://example.com/pic.jpg",
			Plan:        PlanFree,
			Providers: []LinkedProvider{
				{
					ProviderID:  ProviderGoogle,
					Subject:     "google-sub-123",
					Email:       "test@gmail.com",
					DisplayName: "Test Google",
					LinkedAt:    now,
				},
			},
			CreatedAt:   now,
			UpdatedAt:   now,
			LastLoginAt: now,
		}

		data := userToMap(user)

		if data["uid"] != "uid-123" {
			t.Errorf("Expected uid 'uid-123', got %v", data["uid"])
		}
		if data["email"] != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got %v", data["email"])
		}
		if data["displayName"] != "Test User" {
			t.Errorf("Expected displayName 'Test User', got %v", data["displayName"])
		}
		if data["pictureUrl"] != "https://example.com/pic.jpg" {
			t.Errorf("Expected pictureUrl, got %v", data["pictureUrl"])
		}
		if data["plan"] != PlanFree {
			t.Errorf("Expected plan 'free', got %v", data["plan"])
		}
		if data["createdAt"] != now {
			t.Errorf("Expected createdAt %v, got %v", now, data["createdAt"])
		}
		if data["updatedAt"] != now {
			t.Errorf("Expected updatedAt %v, got %v", now, data["updatedAt"])
		}
		if data["lastLoginAt"] != now {
			t.Errorf("Expected lastLoginAt %v, got %v", now, data["lastLoginAt"])
		}

		providers, ok := data["providers"].([]map[string]interface{})
		if !ok {
			t.Fatal("Expected providers to be a slice of maps")
		}
		if len(providers) != 1 {
			t.Fatalf("Expected 1 provider, got %d", len(providers))
		}
		if providers[0]["providerId"] != ProviderGoogle {
			t.Errorf("Expected providerId 'google.com', got %v", providers[0]["providerId"])
		}
		if providers[0]["subject"] != "google-sub-123" {
			t.Errorf("Expected subject 'google-sub-123', got %v", providers[0]["subject"])
		}
	})

	t.Run("converts user without optional fields", func(t *testing.T) {
		user := User{
			ID:          "doc-id",
			UID:         "uid-456",
			Email:       "minimal@example.com",
			DisplayName: "Minimal User",
			Plan:        PlanFree,
			Providers:   []LinkedProvider{},
			CreatedAt:   now,
			UpdatedAt:   now,
			LastLoginAt: now,
		}

		data := userToMap(user)

		if data["uid"] != "uid-456" {
			t.Errorf("Expected uid 'uid-456', got %v", data["uid"])
		}
		// pictureUrl should be empty string or not present
		if url, ok := data["pictureUrl"].(string); ok && url != "" {
			t.Errorf("Expected empty pictureUrl, got %v", url)
		}

		providers, ok := data["providers"].([]map[string]interface{})
		if !ok {
			t.Fatal("Expected providers to be a slice of maps")
		}
		if len(providers) != 0 {
			t.Errorf("Expected 0 providers, got %d", len(providers))
		}
	})

	t.Run("does not include ID in map", func(t *testing.T) {
		user := User{
			ID:          "doc-id",
			UID:         "uid-789",
			Email:       "test@example.com",
			DisplayName: "Test",
			Plan:        PlanFree,
			CreatedAt:   now,
			UpdatedAt:   now,
			LastLoginAt: now,
		}

		data := userToMap(user)

		if _, exists := data["id"]; exists {
			t.Error("ID should not be included in the map (it's the document ID)")
		}
	})

	t.Run("includes Stripe fields when set", func(t *testing.T) {
		subscriptionEndsAt := now.Add(30 * 24 * time.Hour)
		user := User{
			ID:                 "doc-id",
			UID:                "uid-stripe",
			Email:              "stripe@example.com",
			DisplayName:        "Stripe User",
			Plan:               PlanPro,
			StripeCustomerID:   "cus_test123",
			SubscriptionID:     "sub_test456",
			SubscriptionStatus: SubscriptionStatusActive,
			SubscriptionEndsAt: subscriptionEndsAt,
			CreatedAt:          now,
			UpdatedAt:          now,
			LastLoginAt:        now,
		}

		data := userToMap(user)

		if data["stripeCustomerId"] != "cus_test123" {
			t.Errorf("Expected stripeCustomerId 'cus_test123', got %v", data["stripeCustomerId"])
		}
		if data["subscriptionId"] != "sub_test456" {
			t.Errorf("Expected subscriptionId 'sub_test456', got %v", data["subscriptionId"])
		}
		if data["subscriptionStatus"] != SubscriptionStatusActive {
			t.Errorf("Expected subscriptionStatus 'active', got %v", data["subscriptionStatus"])
		}
		if data["subscriptionEndsAt"] != subscriptionEndsAt {
			t.Errorf("Expected subscriptionEndsAt %v, got %v", subscriptionEndsAt, data["subscriptionEndsAt"])
		}
	})

	t.Run("omits Stripe fields when not set", func(t *testing.T) {
		user := User{
			ID:          "doc-id",
			UID:         "uid-no-stripe",
			Email:       "nostripe@example.com",
			DisplayName: "No Stripe User",
			Plan:        PlanFree,
			CreatedAt:   now,
			UpdatedAt:   now,
			LastLoginAt: now,
		}

		data := userToMap(user)

		if _, exists := data["stripeCustomerId"]; exists {
			t.Error("stripeCustomerId should not be included when not set")
		}
		if _, exists := data["subscriptionId"]; exists {
			t.Error("subscriptionId should not be included when not set")
		}
		if _, exists := data["subscriptionStatus"]; exists {
			t.Error("subscriptionStatus should not be included when not set")
		}
		if _, exists := data["subscriptionEndsAt"]; exists {
			t.Error("subscriptionEndsAt should not be included when not set")
		}
	})
}

func TestProviderToMap(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("converts provider with all fields", func(t *testing.T) {
		provider := LinkedProvider{
			ProviderID:  ProviderGoogle,
			Subject:     "sub-123",
			Email:       "test@gmail.com",
			DisplayName: "Test Google",
			LinkedAt:    now,
		}

		data := providerToMap(provider)

		if data["providerId"] != ProviderGoogle {
			t.Errorf("Expected providerId 'google.com', got %v", data["providerId"])
		}
		if data["subject"] != "sub-123" {
			t.Errorf("Expected subject 'sub-123', got %v", data["subject"])
		}
		if data["email"] != "test@gmail.com" {
			t.Errorf("Expected email 'test@gmail.com', got %v", data["email"])
		}
		if data["displayName"] != "Test Google" {
			t.Errorf("Expected displayName 'Test Google', got %v", data["displayName"])
		}
		if data["linkedAt"] != now {
			t.Errorf("Expected linkedAt %v, got %v", now, data["linkedAt"])
		}
	})

	t.Run("converts provider without optional fields", func(t *testing.T) {
		provider := LinkedProvider{
			ProviderID: ProviderPassword,
			Subject:    "password-sub",
			LinkedAt:   now,
		}

		data := providerToMap(provider)

		if data["providerId"] != ProviderPassword {
			t.Errorf("Expected providerId 'password', got %v", data["providerId"])
		}
		// Optional fields should be empty or not present
		if email, ok := data["email"].(string); ok && email != "" {
			t.Errorf("Expected empty email, got %v", email)
		}
	})
}

func TestErrNotFound(t *testing.T) {
	t.Run("error message is correct", func(t *testing.T) {
		if ErrNotFound.Error() != "user not found" {
			t.Errorf("ErrNotFound message = %q, want %q", ErrNotFound.Error(), "user not found")
		}
	})
}

func TestErrDuplicateUID(t *testing.T) {
	t.Run("error message is correct", func(t *testing.T) {
		if ErrDuplicateUID.Error() != "user with this UID already exists" {
			t.Errorf("ErrDuplicateUID message = %q, want %q", ErrDuplicateUID.Error(), "user with this UID already exists")
		}
	})
}

func TestErrProviderExists(t *testing.T) {
	t.Run("error message is correct", func(t *testing.T) {
		if ErrProviderExists.Error() != "provider already linked to user" {
			t.Errorf("ErrProviderExists message = %q, want %q", ErrProviderExists.Error(), "provider already linked to user")
		}
	})
}

func TestErrProviderNotFound(t *testing.T) {
	t.Run("error message is correct", func(t *testing.T) {
		if ErrProviderNotFound.Error() != "provider not found for user" {
			t.Errorf("ErrProviderNotFound message = %q, want %q", ErrProviderNotFound.Error(), "provider not found for user")
		}
	})
}

func TestCollectionName(t *testing.T) {
	t.Run("collection name is correct", func(t *testing.T) {
		if collectionName != "users" {
			t.Errorf("collectionName = %q, want %q", collectionName, "users")
		}
	})
}

func TestUserCopy(t *testing.T) {
	now := time.Now().UTC()

	t.Run("creates independent copy of user", func(t *testing.T) {
		original := User{
			ID:          "doc-id",
			UID:         "uid-123",
			Email:       "test@example.com",
			DisplayName: "Test User",
			PictureURL:  "https://example.com/pic.jpg",
			Plan:        PlanFree,
			Providers: []LinkedProvider{
				{
					ProviderID: ProviderGoogle,
					Subject:    "sub-123",
					LinkedAt:   now,
				},
			},
			CreatedAt:   now,
			UpdatedAt:   now,
			LastLoginAt: now,
		}

		copied := original.Copy()

		// Verify values are equal
		if copied.ID != original.ID {
			t.Errorf("Expected ID %s, got %s", original.ID, copied.ID)
		}
		if copied.UID != original.UID {
			t.Errorf("Expected UID %s, got %s", original.UID, copied.UID)
		}
		if len(copied.Providers) != len(original.Providers) {
			t.Errorf("Expected %d providers, got %d", len(original.Providers), len(copied.Providers))
		}

		// Verify independence (modifying copy doesn't affect original)
		copied.Email = "modified@example.com"
		if original.Email == copied.Email {
			t.Error("Modifying copy should not affect original")
		}

		// Verify providers slice is independent
		copied.Providers[0].ProviderID = "modified"
		if original.Providers[0].ProviderID == "modified" {
			t.Error("Modifying copied providers should not affect original")
		}
	})

	t.Run("handles nil providers", func(t *testing.T) {
		original := User{
			ID:        "doc-id",
			UID:       "uid-123",
			Providers: nil,
		}

		copied := original.Copy()

		if copied.Providers != nil {
			t.Error("Expected nil providers in copy when original has nil")
		}
	})

	t.Run("copies Stripe fields", func(t *testing.T) {
		subscriptionEndsAt := now.Add(30 * 24 * time.Hour)
		original := User{
			ID:                 "doc-id",
			UID:                "uid-stripe",
			Email:              "stripe@example.com",
			Plan:               PlanPro,
			StripeCustomerID:   "cus_test123",
			SubscriptionID:     "sub_test456",
			SubscriptionStatus: SubscriptionStatusActive,
			SubscriptionEndsAt: subscriptionEndsAt,
		}

		copied := original.Copy()

		if copied.StripeCustomerID != original.StripeCustomerID {
			t.Errorf("Expected StripeCustomerID %s, got %s", original.StripeCustomerID, copied.StripeCustomerID)
		}
		if copied.SubscriptionID != original.SubscriptionID {
			t.Errorf("Expected SubscriptionID %s, got %s", original.SubscriptionID, copied.SubscriptionID)
		}
		if copied.SubscriptionStatus != original.SubscriptionStatus {
			t.Errorf("Expected SubscriptionStatus %s, got %s", original.SubscriptionStatus, copied.SubscriptionStatus)
		}
		if !copied.SubscriptionEndsAt.Equal(original.SubscriptionEndsAt) {
			t.Errorf("Expected SubscriptionEndsAt %v, got %v", original.SubscriptionEndsAt, copied.SubscriptionEndsAt)
		}

		// Verify independence
		copied.StripeCustomerID = "cus_modified"
		if original.StripeCustomerID == copied.StripeCustomerID {
			t.Error("Modifying copy should not affect original")
		}
	})
}

func TestLinkedProviderCopy(t *testing.T) {
	now := time.Now().UTC()

	t.Run("creates independent copy of provider", func(t *testing.T) {
		original := LinkedProvider{
			ProviderID:  ProviderGoogle,
			Subject:     "sub-123",
			Email:       "test@gmail.com",
			DisplayName: "Test",
			LinkedAt:    now,
		}

		copied := original.Copy()

		// Verify values are equal
		if copied.ProviderID != original.ProviderID {
			t.Errorf("Expected ProviderID %s, got %s", original.ProviderID, copied.ProviderID)
		}
		if copied.Subject != original.Subject {
			t.Errorf("Expected Subject %s, got %s", original.Subject, copied.Subject)
		}

		// Verify independence
		copied.Email = "modified@gmail.com"
		if original.Email == copied.Email {
			t.Error("Modifying copy should not affect original")
		}
	})
}

func TestFindProviderIndex(t *testing.T) {
	providers := []LinkedProvider{
		{ProviderID: ProviderGoogle, Subject: "google-sub"},
		{ProviderID: ProviderApple, Subject: "apple-sub"},
		{ProviderID: ProviderPassword, Subject: "password-sub"},
	}

	t.Run("finds existing provider", func(t *testing.T) {
		idx := findProviderIndex(providers, ProviderGoogle)
		if idx != 0 {
			t.Errorf("Expected index 0, got %d", idx)
		}

		idx = findProviderIndex(providers, ProviderApple)
		if idx != 1 {
			t.Errorf("Expected index 1, got %d", idx)
		}

		idx = findProviderIndex(providers, ProviderPassword)
		if idx != 2 {
			t.Errorf("Expected index 2, got %d", idx)
		}
	})

	t.Run("returns -1 for non-existing provider", func(t *testing.T) {
		idx := findProviderIndex(providers, "unknown")
		if idx != -1 {
			t.Errorf("Expected index -1, got %d", idx)
		}
	})

	t.Run("handles empty slice", func(t *testing.T) {
		idx := findProviderIndex([]LinkedProvider{}, ProviderGoogle)
		if idx != -1 {
			t.Errorf("Expected index -1, got %d", idx)
		}
	})

	t.Run("handles nil slice", func(t *testing.T) {
		idx := findProviderIndex(nil, ProviderGoogle)
		if idx != -1 {
			t.Errorf("Expected index -1, got %d", idx)
		}
	})
}

func TestMapToProvider(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)

	t.Run("converts map with all fields", func(t *testing.T) {
		data := map[string]interface{}{
			"providerId":  ProviderGoogle,
			"subject":     "sub-123",
			"email":       "test@gmail.com",
			"displayName": "Test Google",
			"linkedAt":    now,
		}

		provider := mapToProvider(data)

		if provider.ProviderID != ProviderGoogle {
			t.Errorf("Expected ProviderID 'google.com', got %s", provider.ProviderID)
		}
		if provider.Subject != "sub-123" {
			t.Errorf("Expected Subject 'sub-123', got %s", provider.Subject)
		}
		if provider.Email != "test@gmail.com" {
			t.Errorf("Expected Email 'test@gmail.com', got %s", provider.Email)
		}
		if provider.DisplayName != "Test Google" {
			t.Errorf("Expected DisplayName 'Test Google', got %s", provider.DisplayName)
		}
		if !provider.LinkedAt.Equal(now) {
			t.Errorf("Expected LinkedAt %v, got %v", now, provider.LinkedAt)
		}
	})

	t.Run("handles missing optional fields", func(t *testing.T) {
		data := map[string]interface{}{
			"providerId": ProviderPassword,
			"subject":    "password-sub",
			"linkedAt":   now,
		}

		provider := mapToProvider(data)

		if provider.ProviderID != ProviderPassword {
			t.Errorf("Expected ProviderID 'password', got %s", provider.ProviderID)
		}
		if provider.Email != "" {
			t.Errorf("Expected empty Email, got %s", provider.Email)
		}
		if provider.DisplayName != "" {
			t.Errorf("Expected empty DisplayName, got %s", provider.DisplayName)
		}
	})

	t.Run("handles empty map", func(t *testing.T) {
		data := map[string]interface{}{}

		provider := mapToProvider(data)

		if provider.ProviderID != "" {
			t.Errorf("Expected empty ProviderID, got %s", provider.ProviderID)
		}
		if provider.Subject != "" {
			t.Errorf("Expected empty Subject, got %s", provider.Subject)
		}
	})

	t.Run("handles wrong type values gracefully", func(t *testing.T) {
		data := map[string]interface{}{
			"providerId":  123,          // wrong type (int instead of string)
			"subject":     true,         // wrong type (bool instead of string)
			"email":       []byte("x"),  // wrong type
			"displayName": 3.14,         // wrong type
			"linkedAt":    "not-a-time", // wrong type
		}

		provider := mapToProvider(data)

		// All fields should be zero values due to type assertion failures
		if provider.ProviderID != "" {
			t.Errorf("Expected empty ProviderID, got %s", provider.ProviderID)
		}
		if provider.Subject != "" {
			t.Errorf("Expected empty Subject, got %s", provider.Subject)
		}
		if provider.Email != "" {
			t.Errorf("Expected empty Email, got %s", provider.Email)
		}
		if provider.DisplayName != "" {
			t.Errorf("Expected empty DisplayName, got %s", provider.DisplayName)
		}
		if !provider.LinkedAt.IsZero() {
			t.Errorf("Expected zero LinkedAt, got %v", provider.LinkedAt)
		}
	})
}

func TestConstants(t *testing.T) {
	t.Run("PlanFree constant", func(t *testing.T) {
		if PlanFree != "free" {
			t.Errorf("PlanFree = %q, want %q", PlanFree, "free")
		}
	})

	t.Run("PlanPro constant", func(t *testing.T) {
		if PlanPro != "pro" {
			t.Errorf("PlanPro = %q, want %q", PlanPro, "pro")
		}
	})

	t.Run("ProviderGoogle constant", func(t *testing.T) {
		if ProviderGoogle != "google.com" {
			t.Errorf("ProviderGoogle = %q, want %q", ProviderGoogle, "google.com")
		}
	})

	t.Run("ProviderApple constant", func(t *testing.T) {
		if ProviderApple != "apple.com" {
			t.Errorf("ProviderApple = %q, want %q", ProviderApple, "apple.com")
		}
	})

	t.Run("ProviderPassword constant", func(t *testing.T) {
		if ProviderPassword != "password" {
			t.Errorf("ProviderPassword = %q, want %q", ProviderPassword, "password")
		}
	})
}

func TestUserStructFields(t *testing.T) {
	now := time.Now().UTC()

	t.Run("User struct can be instantiated with all fields", func(t *testing.T) {
		user := User{
			ID:          "doc-id",
			UID:         "uid-123",
			Email:       "test@example.com",
			DisplayName: "Test User",
			PictureURL:  "https://example.com/pic.jpg",
			Plan:        PlanPro,
			Providers: []LinkedProvider{
				{
					ProviderID:  ProviderGoogle,
					Subject:     "google-sub",
					Email:       "test@gmail.com",
					DisplayName: "Google User",
					LinkedAt:    now,
				},
				{
					ProviderID:  ProviderApple,
					Subject:     "apple-sub",
					Email:       "test@icloud.com",
					DisplayName: "Apple User",
					LinkedAt:    now,
				},
			},
			CreatedAt:   now,
			UpdatedAt:   now,
			LastLoginAt: now,
		}

		if user.ID != "doc-id" {
			t.Errorf("Expected ID 'doc-id', got %s", user.ID)
		}
		if user.UID != "uid-123" {
			t.Errorf("Expected UID 'uid-123', got %s", user.UID)
		}
		if user.Plan != PlanPro {
			t.Errorf("Expected Plan 'pro', got %s", user.Plan)
		}
		if len(user.Providers) != 2 {
			t.Errorf("Expected 2 providers, got %d", len(user.Providers))
		}
	})

	t.Run("User struct can be instantiated with minimal fields", func(t *testing.T) {
		user := User{
			UID:   "uid-minimal",
			Email: "minimal@example.com",
			Plan:  PlanFree,
		}

		if user.ID != "" {
			t.Errorf("Expected empty ID, got %s", user.ID)
		}
		if user.PictureURL != "" {
			t.Errorf("Expected empty PictureURL, got %s", user.PictureURL)
		}
		if user.Providers != nil {
			t.Errorf("Expected nil Providers, got %v", user.Providers)
		}
	})
}

func TestLinkedProviderStructFields(t *testing.T) {
	now := time.Now().UTC()

	t.Run("LinkedProvider struct can be instantiated with all fields", func(t *testing.T) {
		provider := LinkedProvider{
			ProviderID:  ProviderGoogle,
			Subject:     "sub-123",
			Email:       "test@gmail.com",
			DisplayName: "Test User",
			LinkedAt:    now,
		}

		if provider.ProviderID != ProviderGoogle {
			t.Errorf("Expected ProviderID 'google.com', got %s", provider.ProviderID)
		}
		if provider.Subject != "sub-123" {
			t.Errorf("Expected Subject 'sub-123', got %s", provider.Subject)
		}
	})

	t.Run("LinkedProvider struct can be instantiated with minimal fields", func(t *testing.T) {
		provider := LinkedProvider{
			ProviderID: ProviderPassword,
			Subject:    "password-sub",
		}

		if provider.Email != "" {
			t.Errorf("Expected empty Email, got %s", provider.Email)
		}
		if provider.DisplayName != "" {
			t.Errorf("Expected empty DisplayName, got %s", provider.DisplayName)
		}
		if !provider.LinkedAt.IsZero() {
			t.Errorf("Expected zero LinkedAt, got %v", provider.LinkedAt)
		}
	})
}

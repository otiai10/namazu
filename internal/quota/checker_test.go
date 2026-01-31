package quota

import (
	"context"
	"errors"
	"testing"

	"github.com/ayanel/namazu/internal/subscription"
)

// mockSubscriptionRepo is a mock implementation of subscription.Repository for testing
type mockSubscriptionRepo struct {
	subscriptions []subscription.Subscription
	listByUserErr error
}

func (m *mockSubscriptionRepo) List(ctx context.Context) ([]subscription.Subscription, error) {
	return m.subscriptions, nil
}

func (m *mockSubscriptionRepo) ListByUserID(ctx context.Context, userID string) ([]subscription.Subscription, error) {
	if m.listByUserErr != nil {
		return nil, m.listByUserErr
	}
	result := make([]subscription.Subscription, 0)
	for _, sub := range m.subscriptions {
		if sub.UserID == userID {
			result = append(result, sub)
		}
	}
	return result, nil
}

func (m *mockSubscriptionRepo) Create(ctx context.Context, sub subscription.Subscription) (string, error) {
	return "", nil
}

func (m *mockSubscriptionRepo) Get(ctx context.Context, id string) (*subscription.Subscription, error) {
	return nil, nil
}

func (m *mockSubscriptionRepo) Update(ctx context.Context, id string, sub subscription.Subscription) error {
	return nil
}

func (m *mockSubscriptionRepo) Delete(ctx context.Context, id string) error {
	return nil
}

func TestChecker_CanCreateSubscription_FreeUserUnderLimit(t *testing.T) {
	// Free user with 0 subscriptions should be able to create one (limit is 1)
	repo := &mockSubscriptionRepo{
		subscriptions: []subscription.Subscription{},
	}
	checker := NewChecker(repo)

	canCreate, err := checker.CanCreateSubscription(context.Background(), "user1", "free")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !canCreate {
		t.Error("expected canCreate = true, got false")
	}
}

func TestChecker_CanCreateSubscription_FreeUserAtLimit(t *testing.T) {
	// Free user with 1 subscription should NOT be able to create one more (limit is 1)
	repo := &mockSubscriptionRepo{
		subscriptions: []subscription.Subscription{
			{ID: "1", UserID: "user1"},
		},
	}
	checker := NewChecker(repo)

	canCreate, err := checker.CanCreateSubscription(context.Background(), "user1", "free")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if canCreate {
		t.Error("expected canCreate = false for free user at limit, got true")
	}
}

func TestChecker_CanCreateSubscription_FreeUserOverLimit(t *testing.T) {
	// Free user with 2 subscriptions (somehow) should NOT be able to create one more
	repo := &mockSubscriptionRepo{
		subscriptions: []subscription.Subscription{
			{ID: "1", UserID: "user1"},
			{ID: "2", UserID: "user1"},
		},
	}
	checker := NewChecker(repo)

	canCreate, err := checker.CanCreateSubscription(context.Background(), "user1", "free")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if canCreate {
		t.Error("expected canCreate = false for free user over limit, got true")
	}
}

func TestChecker_CanCreateSubscription_ProUserCanCreateMany(t *testing.T) {
	// Pro user with 10 subscriptions should be able to create more (limit is 12)
	subs := make([]subscription.Subscription, 10)
	for i := range subs {
		subs[i] = subscription.Subscription{ID: string(rune('1' + i)), UserID: "user1"}
	}
	repo := &mockSubscriptionRepo{subscriptions: subs}
	checker := NewChecker(repo)

	canCreate, err := checker.CanCreateSubscription(context.Background(), "user1", "pro")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !canCreate {
		t.Error("expected canCreate = true for pro user under limit, got false")
	}
}

func TestChecker_CanCreateSubscription_ProUserAtLimit(t *testing.T) {
	// Pro user with 12 subscriptions should NOT be able to create more
	subs := make([]subscription.Subscription, 12)
	for i := range subs {
		subs[i] = subscription.Subscription{ID: string(rune(i)), UserID: "user1"}
	}
	repo := &mockSubscriptionRepo{subscriptions: subs}
	checker := NewChecker(repo)

	canCreate, err := checker.CanCreateSubscription(context.Background(), "user1", "pro")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if canCreate {
		t.Error("expected canCreate = false for pro user at limit, got true")
	}
}

func TestChecker_CanCreateSubscription_UnknownPlanDefaultsToFree(t *testing.T) {
	// Unknown plan with 1 subscription should NOT be able to create more (uses free limits)
	repo := &mockSubscriptionRepo{
		subscriptions: []subscription.Subscription{
			{ID: "1", UserID: "user1"},
		},
	}
	checker := NewChecker(repo)

	canCreate, err := checker.CanCreateSubscription(context.Background(), "user1", "unknown")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if canCreate {
		t.Error("expected canCreate = false for unknown plan at free limit, got true")
	}
}

func TestChecker_CanCreateSubscription_EmptyPlanDefaultsToFree(t *testing.T) {
	// Empty plan with 1 subscription should NOT be able to create more (uses free limits)
	repo := &mockSubscriptionRepo{
		subscriptions: []subscription.Subscription{
			{ID: "1", UserID: "user1"},
		},
	}
	checker := NewChecker(repo)

	canCreate, err := checker.CanCreateSubscription(context.Background(), "user1", "")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if canCreate {
		t.Error("expected canCreate = false for empty plan at free limit, got true")
	}
}

func TestChecker_CanCreateSubscription_RepositoryError(t *testing.T) {
	expectedErr := errors.New("database connection failed")
	repo := &mockSubscriptionRepo{
		listByUserErr: expectedErr,
	}
	checker := NewChecker(repo)

	canCreate, err := checker.CanCreateSubscription(context.Background(), "user1", "free")

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error to wrap %v, got %v", expectedErr, err)
	}
	if canCreate {
		t.Error("expected canCreate = false when error occurs, got true")
	}
}

func TestChecker_CanCreateSubscription_NewUserNoSubscriptions(t *testing.T) {
	// New user with no subscriptions should be able to create
	repo := &mockSubscriptionRepo{
		subscriptions: []subscription.Subscription{},
	}
	checker := NewChecker(repo)

	canCreate, err := checker.CanCreateSubscription(context.Background(), "newuser", "free")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !canCreate {
		t.Error("expected canCreate = true for new user, got false")
	}
}

func TestChecker_CanCreateSubscription_OnlyCountsUserSubscriptions(t *testing.T) {
	// User1 has 0 subscriptions, User2 has 1 subscription
	// User1 should be able to create (only their own count matters)
	repo := &mockSubscriptionRepo{
		subscriptions: []subscription.Subscription{
			{ID: "1", UserID: "user2"},
		},
	}
	checker := NewChecker(repo)

	canCreate, err := checker.CanCreateSubscription(context.Background(), "user1", "free")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !canCreate {
		t.Error("expected canCreate = true (user1 has 0), got false")
	}
}

func TestNewChecker(t *testing.T) {
	repo := &mockSubscriptionRepo{}
	checker := NewChecker(repo)

	if checker == nil {
		t.Error("expected non-nil checker")
	}
}

package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/nexus-shopping/notification-service/internal/domain"
	pg "github.com/nexus-shopping/notification-service/internal/postgres"
)

func setupDB(t *testing.T) (databaseURL string, cleanup func()) {
	t.Helper()
	ctx := context.Background()

	pgContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase("testdb"),
		postgres.WithUsername("test"),
		postgres.WithPassword("test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second),
		),
	)
	require.NoError(t, err)

	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	err = pg.RunMigrations(ctx, connStr)
	require.NoError(t, err)

	return connStr, func() {
		require.NoError(t, pgContainer.Terminate(ctx))
	}
}

func newRepo(t *testing.T, databaseURL string) *pg.Repository {
	t.Helper()
	ctx := context.Background()

	pool, err := pgxpool.New(ctx, databaseURL)
	require.NoError(t, err)
	t.Cleanup(pool.Close)

	return pg.NewRepository(pool)
}

func validNotification(t *testing.T) domain.Notification {
	t.Helper()
	recipient := mustRecipient(t, `{"email":"cliente@example.com"}`)
	fp := domain.ComputeFingerprint(domain.ChannelEmail, recipient, "Hello", "World", "ref-1", "", "")
	n, err := domain.NewNotification(
		uuid.New(), "key-"+uuid.New().String(), fp,
		domain.ChannelEmail, recipient, "Hello", "World", "ref-1",
		"", "",
		time.Now(),
	)
	require.NoError(t, err)
	return n
}

func mustRecipient(t *testing.T, raw string) domain.Recipient {
	t.Helper()
	r, err := domain.NewRecipient([]byte(raw))
	require.NoError(t, err)
	return r
}

func TestInsert_ReturnsNotification(t *testing.T) {
	url, cleanup := setupDB(t)
	defer cleanup()

	repo := newRepo(t, url)
	n := validNotification(t)

	inserted, err := repo.Insert(context.Background(), n)
	require.NoError(t, err)
	require.Equal(t, n.ID, inserted.ID)
	require.Equal(t, domain.StatusPending, inserted.Status)
}

func TestInsert_SameKeyReturnsExisting(t *testing.T) {
	url, cleanup := setupDB(t)
	defer cleanup()

	repo := newRepo(t, url)
	n := validNotification(t)

	_, err := repo.Insert(context.Background(), n)
	require.NoError(t, err)

	existing, err := repo.Insert(context.Background(), n)
	require.NoError(t, err)
	require.Equal(t, n.ID, existing.ID)
}

func TestFindByID_ReturnsInsertedNotification(t *testing.T) {
	url, cleanup := setupDB(t)
	defer cleanup()

	repo := newRepo(t, url)
	n := validNotification(t)

	_, err := repo.Insert(context.Background(), n)
	require.NoError(t, err)

	found, err := repo.FindByID(context.Background(), n.ID)
	require.NoError(t, err)
	require.Equal(t, n.NotificationKey, found.NotificationKey)
	require.Equal(t, domain.StatusPending, found.Status)
}

func TestFindByNotificationKey_ReturnsInsertedNotification(t *testing.T) {
	url, cleanup := setupDB(t)
	defer cleanup()

	repo := newRepo(t, url)
	n := validNotification(t)

	_, err := repo.Insert(context.Background(), n)
	require.NoError(t, err)

	found, err := repo.FindByNotificationKey(context.Background(), n.NotificationKey)
	require.NoError(t, err)
	require.Equal(t, n.ID, found.ID)
}

func TestClaimBatch_PicksPendingNotifications(t *testing.T) {
	url, cleanup := setupDB(t)
	defer cleanup()

	repo := newRepo(t, url)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		n := validNotification(t)
		_, err := repo.Insert(ctx, n)
		require.NoError(t, err)
	}

	claimed, err := repo.ClaimBatch(ctx, 10, 30*time.Second)
	require.NoError(t, err)
	require.Len(t, claimed, 3)

	for _, n := range claimed {
		require.Equal(t, domain.StatusSending, n.Status)
		require.NotEqual(t, uuid.Nil, n.LeaseToken)
		require.False(t, n.LeaseUntil.IsZero())
		require.Equal(t, 1, n.AttemptCount)
	}
}

func TestClaimBatch_DoesNotClaimAlreadyClaimed(t *testing.T) {
	url, cleanup := setupDB(t)
	defer cleanup()

	repo := newRepo(t, url)
	ctx := context.Background()

	n := validNotification(t)
	_, err := repo.Insert(ctx, n)
	require.NoError(t, err)

	claimed, err := repo.ClaimBatch(ctx, 10, 30*time.Second)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	claimedAgain, err := repo.ClaimBatch(ctx, 10, 30*time.Second)
	require.NoError(t, err)
	require.Len(t, claimedAgain, 0)
}

func TestComplete_WithCorrectToken_Succeeds(t *testing.T) {
	url, cleanup := setupDB(t)
	defer cleanup()

	repo := newRepo(t, url)
	ctx := context.Background()

	n := validNotification(t)
	_, err := repo.Insert(ctx, n)
	require.NoError(t, err)

	claimed, err := repo.ClaimBatch(ctx, 10, 30*time.Second)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	now := time.Now()
	ok, err := repo.Complete(ctx, claimed[0].ID, domain.StatusSent, claimed[0].LeaseToken, now, "")
	require.NoError(t, err)
	require.True(t, ok)

	found, err := repo.FindByID(ctx, n.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusSent, found.Status)
}

func TestComplete_WithWrongToken_Fails(t *testing.T) {
	url, cleanup := setupDB(t)
	defer cleanup()

	repo := newRepo(t, url)
	ctx := context.Background()

	n := validNotification(t)
	_, err := repo.Insert(ctx, n)
	require.NoError(t, err)

	claimed, err := repo.ClaimBatch(ctx, 10, 30*time.Second)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	now := time.Now()
	wrongToken := uuid.New()
	ok, err := repo.Complete(ctx, claimed[0].ID, domain.StatusSent, wrongToken, now, "")
	require.NoError(t, err)
	require.False(t, ok)

	found, err := repo.FindByID(ctx, n.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusSending, found.Status)
}

func TestComplete_WithFailure_SetsFailedStatus(t *testing.T) {
	url, cleanup := setupDB(t)
	defer cleanup()

	repo := newRepo(t, url)
	ctx := context.Background()

	n := validNotification(t)
	_, err := repo.Insert(ctx, n)
	require.NoError(t, err)

	claimed, err := repo.ClaimBatch(ctx, 10, 30*time.Second)
	require.NoError(t, err)
	require.Len(t, claimed, 1)

	now := time.Now()
	ok, err := repo.Complete(ctx, claimed[0].ID, domain.StatusFailed, claimed[0].LeaseToken, now, "provider error")
	require.NoError(t, err)
	require.True(t, ok)

	found, err := repo.FindByID(ctx, n.ID)
	require.NoError(t, err)
	require.Equal(t, domain.StatusFailed, found.Status)
	require.Equal(t, "provider error", found.FailureReason)
}

func TestInsertDelivery_StoresDelivery(t *testing.T) {
	url, cleanup := setupDB(t)
	defer cleanup()

	repo := newRepo(t, url)
	ctx := context.Background()

	n := validNotification(t)
	_, err := repo.Insert(ctx, n)
	require.NoError(t, err)

	delivery := domain.NewDelivery(uuid.New(), n.ID, "dkey-1", 1, time.Now())
	err = repo.InsertDelivery(ctx, delivery)
	require.NoError(t, err)
}

func TestCompleteDeliverySuccess_UpdatesStatus(t *testing.T) {
	url, cleanup := setupDB(t)
	defer cleanup()

	repo := newRepo(t, url)
	ctx := context.Background()

	n := validNotification(t)
	_, err := repo.Insert(ctx, n)
	require.NoError(t, err)

	delivery := domain.NewDelivery(uuid.New(), n.ID, "dkey-2", 1, time.Now())
	err = repo.InsertDelivery(ctx, delivery)
	require.NoError(t, err)

	err = repo.CompleteDeliverySuccess(ctx, delivery.ID, "sent ok", time.Now())
	require.NoError(t, err)
}
package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/nexus-shopping/notification-service/internal/app"
	"github.com/nexus-shopping/notification-service/internal/clock"
	"github.com/nexus-shopping/notification-service/internal/domain"
	"github.com/nexus-shopping/notification-service/internal/email"
)

type fakeNotificationRepo struct {
	mu         sync.Mutex
	data       map[uuid.UUID]domain.Notification
	claimCalls int
}

func newFakeNotificationRepo() *fakeNotificationRepo {
	return &fakeNotificationRepo{data: make(map[uuid.UUID]domain.Notification)}
}

func (r *fakeNotificationRepo) Insert(_ context.Context, n domain.Notification) (app.InsertNotificationResult, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[n.ID] = n
	return app.InsertNotificationResult{Notification: n}, nil
}

func (r *fakeNotificationRepo) ClaimBatch(_ context.Context, batchSize int, leaseDuration time.Duration, now time.Time) ([]domain.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.claimCalls++

	leaseUntil := now.Add(leaseDuration)

	var claimed []domain.Notification
	for id, n := range r.data {
		if n.Status != domain.StatusPending {
			continue
		}
		if len(claimed) >= batchSize {
			break
		}
		token := uuid.New()
		n.Status = domain.StatusSending
		n.LeaseToken = token
		n.LeaseUntil = leaseUntil
		n.AttemptCount++
		n.UpdatedAt = now
		r.data[id] = n
		claimed = append(claimed, n)
	}
	return claimed, nil
}

func (r *fakeNotificationRepo) claimedBatches() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.claimCalls
}

func (r *fakeNotificationRepo) Complete(_ context.Context, id uuid.UUID, status domain.Status, leaseToken uuid.UUID, now time.Time, failureReason string) (bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	n, ok := r.data[id]
	if !ok {
		return false, nil
	}
	if n.Status != domain.StatusSending || n.LeaseToken != leaseToken {
		return false, nil
	}

	n.Status = status
	n.LeaseToken = uuid.Nil
	n.UpdatedAt = now
	if status == domain.StatusSent {
		n.SentAt = now
	}
	if status == domain.StatusFailed {
		n.FailureReason = failureReason
	}
	r.data[id] = n
	return true, nil
}

func (r *fakeNotificationRepo) FindByID(_ context.Context, id uuid.UUID) (domain.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	n, ok := r.data[id]
	if !ok {
		return domain.Notification{}, domain.ErrNotificationNotFound
	}
	return n, nil
}

func (r *fakeNotificationRepo) FindByNotificationKey(_ context.Context, key string) (domain.Notification, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, n := range r.data {
		if n.NotificationKey == key {
			return n, nil
		}
	}
	return domain.Notification{}, domain.ErrNotificationNotFound
}

type fakeDeliveryRepo struct {
	mu   sync.Mutex
	data map[uuid.UUID]domain.Delivery
}

func newFakeDeliveryRepo() *fakeDeliveryRepo {
	return &fakeDeliveryRepo{data: make(map[uuid.UUID]domain.Delivery)}
}

func (r *fakeDeliveryRepo) InsertDelivery(_ context.Context, d domain.Delivery) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data[d.ID] = d
	return nil
}

func (r *fakeDeliveryRepo) CompleteDeliverySuccess(_ context.Context, id uuid.UUID, response string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.data[id]
	if !ok {
		return nil
	}
	d.Status = domain.DeliveryStatusSent
	d.ProviderResponse = response
	d.CompletedAt = now
	r.data[id] = d
	return nil
}

func (r *fakeDeliveryRepo) CompleteDeliveryFailure(_ context.Context, id uuid.UUID, reason string, now time.Time) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	d, ok := r.data[id]
	if !ok {
		return nil
	}
	d.Status = domain.DeliveryStatusFailed
	d.FailureReason = reason
	d.CompletedAt = now
	r.data[id] = d
	return nil
}

func (r *fakeDeliveryRepo) count() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.data)
}

type fakeIDGenerator struct {
	mu   sync.Mutex
	next int
}

func (g *fakeIDGenerator) NewID() uuid.UUID {
	g.mu.Lock()
	g.next++
	n := g.next
	g.mu.Unlock()
	var b [16]byte
	b[14] = byte(n >> 8)
	b[15] = byte(n)
	return uuid.UUID(b)
}

type blockingEmailProvider struct {
	mu           sync.Mutex
	block        chan struct{}
	currentCount int32
	maxObserved  int32
	totalCalls   int32
}

func newBlockingEmailProvider() *blockingEmailProvider {
	return &blockingEmailProvider{block: make(chan struct{}, 100)}
}

func (p *blockingEmailProvider) Send(_ context.Context, _ string, _ string, _ string, _ string) (bool, error) {
	cur := atomic.AddInt32(&p.currentCount, 1)
	for {
		max := atomic.LoadInt32(&p.maxObserved)
		if cur <= max {
			break
		}
		if atomic.CompareAndSwapInt32(&p.maxObserved, max, cur) {
			break
		}
	}
	atomic.AddInt32(&p.totalCalls, 1)
	<-p.block
	atomic.AddInt32(&p.currentCount, -1)
	return true, nil
}

func (p *blockingEmailProvider) unblock(n int) {
	for i := 0; i < n; i++ {
		p.block <- struct{}{}
	}
}

type singleFailProvider struct {
	mu     sync.Mutex
	failed bool
}

func (p *singleFailProvider) Send(_ context.Context, _ string, _ string, _ string, _ string) (bool, error) {
	p.mu.Lock()
	if !p.failed {
		p.failed = true
		p.mu.Unlock()
		return false, nil
	}
	p.mu.Unlock()
	return true, nil
}

type drainProvider struct {
	mu       sync.Mutex
	block    chan struct{}
	allReady chan struct{}
	ready    int
	total    int
}

func newDrainProvider(total int) *drainProvider {
	return &drainProvider{
		block:    make(chan struct{}, total),
		allReady: make(chan struct{}),
		total:    total,
	}
}

func (p *drainProvider) Send(ctx context.Context, _ string, _ string, _ string, _ string) (bool, error) {
	p.mu.Lock()
	p.ready++
	if p.ready == p.total {
		close(p.allReady)
	}
	p.mu.Unlock()

	select {
	case <-ctx.Done():
		return false, ctx.Err()
	case <-p.block:
		return true, nil
	}
}

func (p *drainProvider) unblockAll() {
	for i := 0; i < p.total; i++ {
		p.block <- struct{}{}
	}
}

func pendingNotification(id uuid.UUID, key string, emailAddr string) domain.Notification {
	raw := json.RawMessage(`{"email":"` + emailAddr + `"}`)
	recipient, err := domain.NewRecipient(raw)
	if err != nil {
		panic(err)
	}
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	n, err := domain.NewNotification(id, key, "fp-"+key, domain.ChannelEmail, recipient, "Hello", "World", "ref-"+key, "", "", now)
	if err != nil {
		panic(err)
	}
	return n
}

func nullLogger() *slog.Logger {
	return slog.New(slog.NewJSONHandler(&bytes.Buffer{}, &slog.HandlerOptions{Level: slog.LevelInfo}))
}

func TestWorkerProcessesPendingNotifications(t *testing.T) {
	notifRepo := newFakeNotificationRepo()
	deliveryRepo := newFakeDeliveryRepo()
	emailProv := &email.FakeProvider{Log: nullLogger()}
	fakeClock := clock.NewFake(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	idGen := &fakeIDGenerator{}

	id1 := uuid.New()
	id2 := uuid.New()
	notifRepo.Insert(context.Background(), pendingNotification(id1, "key-1", "a@test.com"))
	notifRepo.Insert(context.Background(), pendingNotification(id2, "key-2", "b@test.com"))

	w := New(WorkerDeps{
		Notifications: notifRepo,
		Deliveries:    deliveryRepo,
		Email:         emailProv,
		Clock:         fakeClock,
		IDs:           idGen,
		Log:           nullLogger(),
	}, WorkerConfig{
		PollInterval:   100 * time.Millisecond,
		BatchSize:      10,
		LeaseDuration:  30 * time.Second,
		MaxConcurrency: 2,
	})

	w.processBatch(context.Background())

	n1, err := notifRepo.FindByID(context.Background(), id1)
	require.NoError(t, err)
	require.Equal(t, domain.StatusSent, n1.Status)

	n2, err := notifRepo.FindByID(context.Background(), id2)
	require.NoError(t, err)
	require.Equal(t, domain.StatusSent, n2.Status)

	require.Equal(t, 2, deliveryRepo.count())
}

func TestWorkerHandlesEmailFailure(t *testing.T) {
	notifRepo := newFakeNotificationRepo()
	deliveryRepo := newFakeDeliveryRepo()
	fakeClock := clock.NewFake(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	idGen := &fakeIDGenerator{}

	id1 := uuid.New()
	id2 := uuid.New()
	notifRepo.Insert(context.Background(), pendingNotification(id1, "key-1", "a@test.com"))
	notifRepo.Insert(context.Background(), pendingNotification(id2, "key-2", "b@test.com"))

	emailProv := &singleFailProvider{}

	w := New(WorkerDeps{
		Notifications: notifRepo,
		Deliveries:    deliveryRepo,
		Email:         emailProv,
		Clock:         fakeClock,
		IDs:           idGen,
		Log:           nullLogger(),
	}, WorkerConfig{
		PollInterval:   100 * time.Millisecond,
		BatchSize:      10,
		LeaseDuration:  30 * time.Second,
		MaxConcurrency: 2,
	})

	w.processBatch(context.Background())

	n1, err := notifRepo.FindByID(context.Background(), id1)
	require.NoError(t, err)
	n2, err := notifRepo.FindByID(context.Background(), id2)
	require.NoError(t, err)

	require.True(t, n1.Status.IsTerminal())
	require.True(t, n2.Status.IsTerminal())

	failed := 0
	if n1.Status == domain.StatusFailed {
		failed++
		require.Contains(t, n1.FailureReason, "send failed")
	}
	if n2.Status == domain.StatusFailed {
		failed++
		require.Contains(t, n2.FailureReason, "send failed")
	}
	require.Equal(t, 1, failed, "exactly one notification should fail")

	require.Equal(t, 2, deliveryRepo.count())
}

func TestWorkerRespectsMaxConcurrency(t *testing.T) {
	notifRepo := newFakeNotificationRepo()
	deliveryRepo := newFakeDeliveryRepo()
	fakeClock := clock.NewFake(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	idGen := &fakeIDGenerator{}

	const notificationCount = 10
	const maxConcurrency = 2

	for i := 0; i < notificationCount; i++ {
		key := "key-" + string(rune('a'+i))
		notifRepo.Insert(context.Background(), pendingNotification(uuid.New(), key, "test@test.com"))
	}

	emailProv := newBlockingEmailProvider()

	w := New(WorkerDeps{
		Notifications: notifRepo,
		Deliveries:    deliveryRepo,
		Email:         emailProv,
		Clock:         fakeClock,
		IDs:           idGen,
		Log:           nullLogger(),
	}, WorkerConfig{
		PollInterval:   100 * time.Millisecond,
		BatchSize:      notificationCount,
		LeaseDuration:  30 * time.Second,
		MaxConcurrency: maxConcurrency,
	})

	done := make(chan struct{})
	go func() {
		w.processBatch(context.Background())
		close(done)
	}()

	require.Eventually(t, func() bool {
		return atomic.LoadInt32(&emailProv.totalCalls) >= maxConcurrency
	}, 500*time.Millisecond, time.Millisecond)

	maxObserved := atomic.LoadInt32(&emailProv.maxObserved)
	require.LessOrEqual(t, maxObserved, int32(maxConcurrency))
	require.Equal(t, int32(maxConcurrency), maxObserved)

	emailProv.unblock(notificationCount)

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("processBatch did not complete")
	}

	require.Equal(t, int32(notificationCount), atomic.LoadInt32(&emailProv.totalCalls))
}

func TestWorkerDrainsOnShutdown(t *testing.T) {
	notifRepo := newFakeNotificationRepo()
	deliveryRepo := newFakeDeliveryRepo()
	fakeClock := clock.NewFake(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	idGen := &fakeIDGenerator{}

	const notificationCount = 5

	keys := make([]string, notificationCount)
	for i := 0; i < notificationCount; i++ {
		key := "key-" + string(rune('a'+i))
		keys[i] = key
		notifRepo.Insert(context.Background(), pendingNotification(uuid.New(), key, "test@test.com"))
	}

	drainProv := newDrainProvider(notificationCount)

	w := New(WorkerDeps{
		Notifications: notifRepo,
		Deliveries:    deliveryRepo,
		Email:         drainProv,
		Clock:         fakeClock,
		IDs:           idGen,
		Log:           nullLogger(),
	}, WorkerConfig{
		PollInterval:   100 * time.Millisecond,
		BatchSize:      notificationCount,
		LeaseDuration:  30 * time.Second,
		MaxConcurrency: notificationCount,
	})

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		w.processBatch(ctx)
		close(done)
	}()

	<-drainProv.allReady
	cancel()

	drainProv.unblockAll()

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("processBatch did not complete")
	}

	for _, key := range keys {
		n, err := notifRepo.FindByNotificationKey(context.Background(), key)
		require.NoError(t, err)
		require.True(t, n.Status.IsTerminal(), "notification %s should be terminal, got %s", key, n.Status)
	}
}

func TestWorkerDoesNotClaimAfterCancellation(t *testing.T) {
	notifRepo := newFakeNotificationRepo()
	deliveryRepo := newFakeDeliveryRepo()
	fakeClock := clock.NewFake(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	notifRepo.Insert(context.Background(), pendingNotification(uuid.New(), "key-1", "test@test.com"))

	w := New(WorkerDeps{
		Notifications: notifRepo,
		Deliveries:    deliveryRepo,
		Email:         &email.FakeProvider{Log: nullLogger()},
		Clock:         fakeClock,
		IDs:           &fakeIDGenerator{},
		Log:           nullLogger(),
	}, WorkerConfig{
		PollInterval:   time.Hour,
		BatchSize:      1,
		LeaseDuration:  30 * time.Second,
		MaxConcurrency: 1,
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	require.NoError(t, w.Run(ctx))
	require.Equal(t, 0, notifRepo.claimedBatches())
}

func TestWorkerLogsWithoutSensitiveData(t *testing.T) {
	notifRepo := newFakeNotificationRepo()
	deliveryRepo := newFakeDeliveryRepo()
	fakeClock := clock.NewFake(time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))
	idGen := &fakeIDGenerator{}

	id := uuid.New()
	notifRepo.Insert(context.Background(), pendingNotification(id, "key-1", "secret@private.com"))

	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	w := New(WorkerDeps{
		Notifications: notifRepo,
		Deliveries:    deliveryRepo,
		Email:         &email.FakeProvider{Log: logger},
		Clock:         fakeClock,
		IDs:           idGen,
		Log:           logger,
	}, WorkerConfig{
		PollInterval:   100 * time.Millisecond,
		BatchSize:      10,
		LeaseDuration:  30 * time.Second,
		MaxConcurrency: 2,
	})

	w.processBatch(context.Background())
	w.log.Info("worker stopped")

	output := buf.String()
	lines := bytes.Split(bytes.TrimSpace([]byte(output)), []byte("\n"))
	require.NotEmpty(t, lines)

	for _, line := range lines {
		var entry map[string]interface{}
		require.NoError(t, json.Unmarshal(line, &entry), "invalid JSON log line: %s", line)

		msg, _ := entry["msg"].(string)
		require.NotContains(t, msg, "secret@private.com", "log message must not contain email: %s", msg)
		require.NotContains(t, msg, "Hello", "log message must not contain subject: %s", msg)
		require.NotContains(t, msg, "World", "log message must not contain body: %s", msg)

		require.NotContains(t, line, "subject", "log line must not contain subject: %s", line)
		require.NotContains(t, line, "body", "log line must not contain body: %s", line)
		require.NotContains(t, line, "secret@private.com", "log line must not contain email: %s", line)
	}
}

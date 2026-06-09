package background

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"status-incident/internal/application"
	"status-incident/internal/domain"
)

// errStubGetAll is returned by the stub repo to exercise error branches.
var errStubGetAll = errors.New("stub get-all failure")

// --- Stub domain implementations used to build a real HeartbeatService ---

// stubDependencyRepository implements domain.DependencyRepository. It records
// how many times GetAllWithHeartbeat is invoked and signals on a channel so
// tests can wait deterministically instead of sleeping.
type stubDependencyRepository struct {
	mu        sync.Mutex
	deps      []*domain.Dependency
	called    int32
	callCh    chan struct{}
	updateErr error
	getAllErr error
}

func newStubDependencyRepository() *stubDependencyRepository {
	return &stubDependencyRepository{
		callCh: make(chan struct{}, 1024),
	}
}

func (r *stubDependencyRepository) Create(ctx context.Context, dep *domain.Dependency) error {
	return nil
}

func (r *stubDependencyRepository) GetByID(ctx context.Context, id int64) (*domain.Dependency, error) {
	return nil, nil
}

func (r *stubDependencyRepository) GetBySystemID(ctx context.Context, systemID int64) ([]*domain.Dependency, error) {
	return nil, nil
}

func (r *stubDependencyRepository) GetAllWithHeartbeat(ctx context.Context) ([]*domain.Dependency, error) {
	atomic.AddInt32(&r.called, 1)
	// Non-blocking signal; channel is buffered.
	select {
	case r.callCh <- struct{}{}:
	default:
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.getAllErr != nil {
		return nil, r.getAllErr
	}
	return r.deps, nil
}

func (r *stubDependencyRepository) Update(ctx context.Context, dep *domain.Dependency) error {
	return r.updateErr
}

func (r *stubDependencyRepository) Delete(ctx context.Context, id int64) error {
	return nil
}

// stubStatusLogRepository implements domain.StatusLogRepository.
type stubStatusLogRepository struct{}

func (r *stubStatusLogRepository) Create(ctx context.Context, log *domain.StatusLog) error {
	return nil
}

func (r *stubStatusLogRepository) GetBySystemID(ctx context.Context, systemID int64, limit int) ([]*domain.StatusLog, error) {
	return nil, nil
}

func (r *stubStatusLogRepository) GetByDependencyID(ctx context.Context, dependencyID int64, limit int) ([]*domain.StatusLog, error) {
	return nil, nil
}

func (r *stubStatusLogRepository) GetAll(ctx context.Context, limit int) ([]*domain.StatusLog, error) {
	return nil, nil
}

func (r *stubStatusLogRepository) GetByTimeRange(ctx context.Context, start, end time.Time) ([]*domain.StatusLog, error) {
	return nil, nil
}

func (r *stubStatusLogRepository) GetSystemLogsByTimeRange(ctx context.Context, systemID int64, start, end time.Time) ([]*domain.StatusLog, error) {
	return nil, nil
}

func (r *stubStatusLogRepository) GetDependencyLogsByTimeRange(ctx context.Context, dependencyID int64, start, end time.Time) ([]*domain.StatusLog, error) {
	return nil, nil
}

// stubHealthChecker implements domain.HealthChecker.
type stubHealthChecker struct {
	result domain.HealthCheckResult
}

func (c *stubHealthChecker) Check(ctx context.Context, url string) (bool, int64, error) {
	return c.result.Healthy, c.result.LatencyMs, c.result.Error
}

func (c *stubHealthChecker) CheckWithConfig(ctx context.Context, config domain.HeartbeatConfig) domain.HealthCheckResult {
	return c.result
}

// newTestService builds a real HeartbeatService backed by the stub repository.
func newTestService(depRepo *stubDependencyRepository) *application.HeartbeatService {
	logRepo := &stubStatusLogRepository{}
	checker := &stubHealthChecker{
		result: domain.HealthCheckResult{Healthy: true, LatencyMs: 10, StatusCode: 200},
	}
	return application.NewHeartbeatService(depRepo, logRepo, checker)
}

// waitForCalls blocks until the stub repo has been called at least n times, or
// the deadline elapses. Returns true on success.
func waitForCalls(repo *stubDependencyRepository, n int, timeout time.Duration) bool {
	deadline := time.After(timeout)
	for atomic.LoadInt32(&repo.called) < int32(n) {
		select {
		case <-repo.callCh:
		case <-deadline:
			return atomic.LoadInt32(&repo.called) >= int32(n)
		}
	}
	return true
}

// --- Tests ---

func TestNewHeartbeatWorker(t *testing.T) {
	repo := newStubDependencyRepository()
	service := newTestService(repo)

	w := NewHeartbeatWorker(service, time.Minute)
	if w == nil {
		t.Fatal("expected non-nil worker")
	}
	if w.service != service {
		t.Error("expected service to be set")
	}
	if w.interval != time.Minute {
		t.Errorf("expected interval 1m, got %v", w.interval)
	}
	if w.stop == nil {
		t.Error("expected stop channel to be initialized")
	}
	if w.done == nil {
		t.Error("expected done channel to be initialized")
	}
}

// TestHeartbeatWorker_RunsImmediatelyAndStop verifies the worker invokes the
// service immediately on Start and that Stop cleanly shuts it down.
func TestHeartbeatWorker_RunsImmediatelyAndStop(t *testing.T) {
	repo := newStubDependencyRepository()
	service := newTestService(repo)

	// Large interval so only the immediate check fires before we stop.
	w := NewHeartbeatWorker(service, time.Hour)
	w.Start(context.Background())

	if !waitForCalls(repo, 1, 2*time.Second) {
		t.Fatal("expected at least one immediate check call")
	}

	w.Stop()

	// After Stop, the done channel must be closed (Stop already waited on it).
	if got := atomic.LoadInt32(&repo.called); got < 1 {
		t.Errorf("expected >=1 call, got %d", got)
	}
}

// TestHeartbeatWorker_TickLoop verifies the ticker fires repeated checks with a
// tiny interval.
func TestHeartbeatWorker_TickLoop(t *testing.T) {
	repo := newStubDependencyRepository()
	service := newTestService(repo)

	w := NewHeartbeatWorker(service, 5*time.Millisecond)
	w.Start(context.Background())

	// 1 immediate + several ticks.
	if !waitForCalls(repo, 4, 2*time.Second) {
		t.Fatalf("expected >=4 calls from tick loop, got %d", atomic.LoadInt32(&repo.called))
	}

	w.Stop()
}

// TestHeartbeatWorker_ContextCancel verifies the worker exits when the context
// is cancelled, closing the done channel.
func TestHeartbeatWorker_ContextCancel(t *testing.T) {
	repo := newStubDependencyRepository()
	service := newTestService(repo)

	ctx, cancel := context.WithCancel(context.Background())
	w := NewHeartbeatWorker(service, time.Hour)
	w.Start(ctx)

	if !waitForCalls(repo, 1, 2*time.Second) {
		t.Fatal("expected immediate check before cancel")
	}

	cancel()

	// run() closes done on exit; wait for it to confirm the ctx.Done branch ran.
	select {
	case <-w.done:
		// Worker exited via context cancellation.
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not exit after context cancellation")
	}
}

// TestHeartbeatWorker_ContextCancelDuringTick exercises the ctx.Done branch
// while the ticker is also active.
func TestHeartbeatWorker_ContextCancelDuringTick(t *testing.T) {
	repo := newStubDependencyRepository()
	service := newTestService(repo)

	ctx, cancel := context.WithCancel(context.Background())
	w := NewHeartbeatWorker(service, 5*time.Millisecond)
	w.Start(ctx)

	if !waitForCalls(repo, 2, 2*time.Second) {
		t.Fatal("expected multiple checks before cancel")
	}

	cancel()

	select {
	case <-w.done:
	case <-time.After(2 * time.Second):
		t.Fatal("worker did not exit after context cancellation during tick loop")
	}
}

// TestHeartbeatWorker_CheckServiceError exercises the error-logging branch in
// check() when the underlying service returns an error.
func TestHeartbeatWorker_CheckServiceError(t *testing.T) {
	repo := newStubDependencyRepository()
	// Force CheckAllDependencies to return a wrapped error.
	repo.getAllErr = errStubGetAll
	service := newTestService(repo)

	w := NewHeartbeatWorker(service, time.Hour)
	w.Start(context.Background())

	if !waitForCalls(repo, 1, 2*time.Second) {
		t.Fatal("expected the worker to attempt a check despite the service error")
	}

	w.Stop()
}

// TestHeartbeatWorker_CheckWithRealDependency drives the check() path with an
// actual dependency that needs checking, ensuring the service does real work
// (not just an empty list) and that the worker's per-check timeout context is
// honored.
func TestHeartbeatWorker_CheckWithRealDependency(t *testing.T) {
	repo := newStubDependencyRepository()
	dep, err := domain.NewDependency(1, "Redis", "Cache")
	if err != nil {
		t.Fatalf("failed to build dependency: %v", err)
	}
	dep.ID = 1
	if err := dep.SetHeartbeatConfig(domain.HeartbeatConfig{
		URL:      "https://redis.example.com/health",
		Interval: 60,
	}); err != nil {
		t.Fatalf("failed to set heartbeat config: %v", err)
	}
	// LastCheck zero => NeedsCheck() true.
	repo.deps = []*domain.Dependency{dep}

	service := newTestService(repo)

	w := NewHeartbeatWorker(service, time.Hour)
	w.Start(context.Background())

	if !waitForCalls(repo, 1, 2*time.Second) {
		t.Fatal("expected the worker to check the dependency")
	}

	w.Stop()

	// The healthy check should have recorded a success on the dependency.
	if dep.LastLatency != 10 {
		t.Errorf("expected dependency last latency 10, got %d", dep.LastLatency)
	}
}

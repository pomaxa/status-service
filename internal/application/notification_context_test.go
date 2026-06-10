package application

import (
	"context"
	"status-incident/internal/domain"
	"sync/atomic"
	"testing"
)

// ctxObservingWebhookRepo mimics a real DB driver that honors context
// cancellation: GetEnabled returns the context error if the context is already
// cancelled, otherwise it records that it ran. This lets us prove that
// NotifyStatusChange detaches from the caller's (request/worker) context, which
// is always cancelled by the time the detached notification goroutine runs.
type ctxObservingWebhookRepo struct {
	served int32
}

func (r *ctxObservingWebhookRepo) Create(ctx context.Context, w *domain.Webhook) error { return nil }
func (r *ctxObservingWebhookRepo) GetByID(ctx context.Context, id int64) (*domain.Webhook, error) {
	return nil, nil
}
func (r *ctxObservingWebhookRepo) GetAll(ctx context.Context) ([]*domain.Webhook, error) {
	return nil, nil
}
func (r *ctxObservingWebhookRepo) GetEnabled(ctx context.Context) ([]*domain.Webhook, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	atomic.AddInt32(&r.served, 1)
	return nil, nil
}
func (r *ctxObservingWebhookRepo) Update(ctx context.Context, w *domain.Webhook) error { return nil }
func (r *ctxObservingWebhookRepo) Delete(ctx context.Context, id int64) error          { return nil }

// TestNotificationService_NotifyStatusChange_DetachesContext guards against a
// regression where NotifyStatusChange was launched as `go NotifyStatusChange(ctx, ...)`
// with the HTTP request / heartbeat worker context. That context is cancelled as
// soon as the originating request returns, so the webhook lookup failed with
// "context canceled" and notifications were silently dropped.
func TestNotificationService_NotifyStatusChange_DetachesContext(t *testing.T) {
	repo := &ctxObservingWebhookRepo{}
	s := NewNotificationService(repo, NewMockSystemRepository(), NewMockDependencyRepository())

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // simulate the caller's request/worker context already being done

	sysID := int64(1)
	s.NotifyStatusChange(ctx, &domain.StatusLog{
		SystemID:  &sysID,
		OldStatus: domain.StatusGreen,
		NewStatus: domain.StatusRed,
		Source:    domain.SourcePropagation,
	})

	if got := atomic.LoadInt32(&repo.served); got != 1 {
		t.Fatalf("GetEnabled ran with a live context %d time(s), want 1: "+
			"NotifyStatusChange must detach from the caller's cancelled context", got)
	}
}

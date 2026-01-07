package background

import (
	"context"
	"log"
	"status-incident/internal/application"
	"time"
)

// HeartbeatWorker runs periodic health checks
type HeartbeatWorker struct {
	service  *application.HeartbeatService
	interval time.Duration
	stop     chan struct{}
	done     chan struct{}
}

// NewHeartbeatWorker creates a new heartbeat worker
func NewHeartbeatWorker(service *application.HeartbeatService, interval time.Duration) *HeartbeatWorker {
	return &HeartbeatWorker{
		service:  service,
		interval: interval,
		stop:     make(chan struct{}),
		done:     make(chan struct{}),
	}
}

// Start begins the heartbeat checking loop
func (w *HeartbeatWorker) Start(ctx context.Context) {
	go w.run(ctx)
}

// Stop gracefully stops the worker
func (w *HeartbeatWorker) Stop() {
	close(w.stop)
	<-w.done
}

func (w *HeartbeatWorker) run(ctx context.Context) {
	defer close(w.done)

	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Run immediately on start
	w.check(ctx)

	for {
		select {
		case <-ticker.C:
			w.check(ctx)
		case <-w.stop:
			log.Println("Heartbeat worker stopping...")
			return
		case <-ctx.Done():
			log.Println("Heartbeat worker context cancelled...")
			return
		}
	}
}

func (w *HeartbeatWorker) check(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := w.service.CheckAllDependencies(checkCtx); err != nil {
		log.Printf("Heartbeat check error: %v", err)
	}
}

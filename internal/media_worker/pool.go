package media_worker

import (
	"context"
	"sync"
	"time"

	"github.com/ferdn4ndo/userver-filemgr/internal/data"
	"github.com/ferdn4ndo/userver-filemgr/internal/media_processor"
	"github.com/ferdn4ndo/userver-filemgr/lib"
	"go.uber.org/fx"
)

// Pool runs PostgreSQL-queued media jobs in goroutines.
type Pool struct {
	env  lib.Env
	db   *data.DB
	proc *media_processor.Processor
	log  lib.Logger

	stop chan struct{}
	wg   sync.WaitGroup
}

// NewPool constructs a worker pool (started via fx lifecycle).
func NewPool(env lib.Env, db *data.DB, proc *media_processor.Processor, log lib.Logger) *Pool {
	return &Pool{env: env, db: db, proc: proc, log: log}
}

// Register attaches start/stop hooks to the Fx lifecycle.
func Register(lc fx.Lifecycle, p *Pool) {
	lc.Append(fx.Hook{
		OnStart: p.Start,
		OnStop:  p.Stop,
	})
}

// Start launches worker goroutines.
func (p *Pool) Start(_ context.Context) error {
	if !p.env.MediaProcessingEnabled || p.env.MediaWorkerCount <= 0 {
		p.log.Info("media worker disabled (MEDIA_PROCESSING_ENABLED or MEDIA_WORKER_COUNT)")
		return nil
	}
	p.stop = make(chan struct{})
	for i := 0; i < p.env.MediaWorkerCount; i++ {
		p.wg.Add(1)
		go p.loop(i)
	}
	p.log.Infow("media worker started", "workers", p.env.MediaWorkerCount)
	return nil
}

// Stop waits for workers to finish after closing the stop channel.
func (p *Pool) Stop(_ context.Context) error {
	if p.stop != nil {
		close(p.stop)
		p.wg.Wait()
	}
	return nil
}

func (p *Pool) loop(workerID int) {
	defer p.wg.Done()
	tick := time.NewTicker(time.Duration(p.env.MediaWorkerPollMS) * time.Millisecond)
	defer tick.Stop()
	for {
		select {
		case <-p.stop:
			return
		default:
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
		job, err := p.db.ClaimMediaJob(ctx)
		if err != nil {
			p.log.Debugw("media claim error", "worker", workerID, "err", err.Error())
			cancel()
			select {
			case <-p.stop:
				return
			case <-tick.C:
			}
			continue
		}
		if job == nil {
			cancel()
			select {
			case <-p.stop:
				return
			case <-tick.C:
			}
			continue
		}
		if err := p.proc.Process(ctx, job); err != nil {
			_ = p.db.FailMediaJob(context.Background(), job.ID, err.Error())
			p.log.Warnw("media job failed", "worker", workerID, "job", job.ID.String(), "err", err.Error())
		} else {
			_ = p.db.CompleteMediaJob(context.Background(), job.ID)
		}
		cancel()
	}
}

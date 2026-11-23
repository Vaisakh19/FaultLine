package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)


var ErrInvalidSchedulerInterval = errors.New("scheduler interval must be positive")

func runScheduler(ctx context.Context, dbpool *pgxpool.Pool, interval time.Duration) error {
	if interval <= 0 {
		return ErrInvalidSchedulerInterval
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	processNextJob(ctx, dbpool)

	for {
		select {
		case <-ctx.Done():
			return nil

		case <-ticker.C:
			processNextJob(ctx, dbpool)
		}
	}
}


func processNextJob(ctx context.Context, dbpool *pgxpool.Pool) {
	for {
		if ctx.Err() != nil {
			return
		}

		job, err := claimNextJob(ctx, dbpool)

		if err != nil {
			// Cancellation during the database operation is normal.
			if ctx.Err() != nil {
				return
			}

			if errors.Is(err, pgx.ErrNoRows) {
				return
			}

			log.Printf("scheduler: failed to claim job: %v", err)
			return
		}

		log.Printf("scheduler: claimed job %d", job.ID)
	}
}

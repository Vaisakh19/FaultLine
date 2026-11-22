package main

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"time"
)

func runScheduler(ctx context.Context, dbpool *pgxpool.Pool, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	processNextJob(dbpool)
	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			processNextJob(dbpool)
		}
	}
}
func processNextJob(dbpool *pgxpool.Pool) {
    for {
        job, err := claimNextJob(dbpool)

        if err != nil {
            if errors.Is(err, pgx.ErrNoRows) {
                return
            }

            log.Printf("scheduler: failed to claim job: %v", err)
            return
        }

        log.Printf("scheduler: claimed job %d", job.ID)
    }
}

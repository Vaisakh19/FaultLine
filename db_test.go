package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"
	"time"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func setupTestDB(t *testing.T) *pgxpool.Pool {
	connStr := os.Getenv("DATABASE_URL")

	dbURL, err := url.Parse(connStr)
	if err != nil {
		t.Fatalf("invalid database URL: %v", err)
	}

	dbURL.Path = "/faultline_test"

	dbpool, err := pgxpool.New(context.Background(), dbURL.String())
	if err != nil {
		t.Fatalf("could not connect to test database: %v", err)
	}

	err = dbpool.Ping(context.Background())
	if err != nil {
		dbpool.Close()
		t.Fatalf("could not ping test database: %v", err)
	}

	t.Cleanup(func() {
		dbpool.Close()
	})

	return dbpool
}
func TestUpdateJobStatusSuccess(t *testing.T) {
	dbpool := setupTestDB(t)

	job := Job{
		Name:     "Status transition test",
		Priority: 3,
		Status:   JobStatusQueued,
	}

	err := insertJob(dbpool, &job)
	if err != nil {
		t.Fatalf("failed to insert job: %v", err)
	}

	t.Cleanup(func() {
		_, err := dbpool.Exec(
			context.Background(),
			"DELETE FROM jobs WHERE id = $1",
			job.ID,
		)
		if err != nil {
			t.Errorf("cleanup failed: %v", err)
		}
	})

	// First transition: QUEUED -> ASSIGNED
	err = updateJobStatus(dbpool, job.ID, JobStatusAssigned)
	if err != nil {
		t.Fatalf("failed to assign job: %v", err)
	}

	// Second transition: ASSIGNED -> RUNNING
	err = updateJobStatus(dbpool, job.ID, JobStatusRunning)
	if err != nil {
		t.Fatalf("failed to start job: %v", err)
	}

	updatedJob, err := getJobByID(dbpool, job.ID)
	if err != nil {
		t.Fatalf("failed to retrieve job: %v", err)
	}

	if updatedJob.Status != JobStatusRunning {
		t.Errorf(
			"expected status %s, got %s",
			JobStatusRunning,
			updatedJob.Status,
		)
	}
}
func TestUpdateJobStatusInvalidTransition(t *testing.T) {
	dbpool := setupTestDB(t)

	job := Job{
		Name:     "Invalid transition test",
		Priority: 2,
		Status:   JobStatusQueued,
	}

	err := insertJob(dbpool, &job)
	if err != nil {
		t.Fatalf("failed to insert job: %v", err)
	}

	t.Cleanup(func() {
		_, err := dbpool.Exec(
			context.Background(),
			"DELETE FROM jobs WHERE id = $1",
			job.ID,
		)
		if err != nil {
			t.Errorf("cleanup failed: %v", err)
		}
	})

	err = updateJobStatus(dbpool, job.ID, JobStatusCompleted)

	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf(
			"expected ErrInvalidTransition, got %v",
			err,
		)
	}

	unchangedJob, err := getJobByID(dbpool, job.ID)
	if err != nil {
		t.Fatalf("failed to retrieve job: %v", err)
	}

	if unchangedJob.Status != JobStatusQueued {
		t.Errorf(
			"expected status PENDING, got %s",
			unchangedJob.Status,
		)
	}
}
func TestUpdateJobStatusNotFound(t *testing.T) {
	dbpool := setupTestDB(t)

	err := updateJobStatus(dbpool, -1, JobStatusRunning)

	if !errors.Is(err, ErrJobNotFound) {
		t.Errorf(
			"expected ErrJobNotFound, got %v",
			err,
		)
	}
}
func TestGetNextJob(t *testing.T) {
	dbpool := setupTestDB(t)

	jobs := []Job{
		{
			Name:     "Email Report",
			Priority: 2,
			Status:   JobStatusQueued,
		},
		{
			Name:     "Generate Invoice",
			Priority: 5,
			Status:   JobStatusQueued,
		},
		{
			Name:     "Backup Data",
			Priority: 3,
			Status:   JobStatusQueued,
		},
		{
			Name:     "Send Notification",
			Priority: 5,
			Status:   JobStatusQueued,
		},
	}

	// Clean up only jobs created by this test.
	t.Cleanup(func() {
		for _, job := range jobs {
			if job.ID == 0 {
				continue
			}

			_, err := dbpool.Exec(
				context.Background(),
				"DELETE FROM jobs WHERE id = $1",
				job.ID,
			)

			if err != nil {
				t.Errorf("cleanup failed for job %d: %v", job.ID, err)
			}
		}
	})

	// Insert test jobs.
	for i := range jobs {
		err := insertJob(dbpool, &jobs[i])
		if err != nil {
			t.Fatalf("failed to insert test job: %v", err)
		}
	}

	// Test highest priority and oldest job selection.
	nextJob, err := getNextJob(dbpool)
	if err != nil {
		t.Fatalf("failed to get next job: %v", err)
	}

	t.Logf(
		"Selected Job: ID=%d, Name=%s, Priority=%d, Status=%s",
		nextJob.ID,
		nextJob.Name,
		nextJob.Priority,
		nextJob.Status,
	)

	if nextJob.ID != jobs[1].ID {
		t.Errorf(
			"expected job ID %d, got %d",
			jobs[1].ID,
			nextJob.ID,
		)
	}

	if nextJob.Name != "Generate Invoice" {
		t.Errorf(
			"expected Generate Invoice, got %s",
			nextJob.Name,
		)
	}

	// Remove only the queued jobs created by this test.
	for _, job := range jobs {
		_, err := dbpool.Exec(
			context.Background(),
			"DELETE FROM jobs WHERE id = $1",
			job.ID,
		)
		if err != nil {
			t.Fatalf("failed to delete test job: %v", err)
		}
	}

	// Test when no queued jobs exist.
	nonQueuedJob := Job{
		Name:     "Completed Job",
		Priority: 5,
		Status:   JobStatusCompleted,
	}

	err = insertJob(dbpool, &nonQueuedJob)
	if err != nil {
		t.Fatalf("failed to insert completed job: %v", err)
	}

	// Test empty queue.
	_, err = getNextJob(dbpool)
	fmt.Printf(
		"Selected Job: ID=%d, Name=%s, Priority=%d, Status=%s\n",
		nextJob.ID,
		nextJob.Name,
		nextJob.Priority,
		nextJob.Status,
	)
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Errorf("expected pgx.ErrNoRows, got %v", err)
	}

	// Clean up completed job.
	_, err = dbpool.Exec(
		context.Background(),
		"DELETE FROM jobs WHERE id = $1",
		nonQueuedJob.ID,
	)
	if err != nil {
		t.Errorf("cleanup failed: %v", err)
	}
}
func TestClaimNextJob(t *testing.T) {
	dbpool := setupTestDB(t)

	jobs := []*Job{
		{Name: "Low Priority", Priority: 2, Status: JobStatusQueued},
		{Name: "High Priority", Priority: 5, Status: JobStatusQueued},
		{Name: "Medium Priority", Priority: 3, Status: JobStatusQueued},
	}

	for _, job := range jobs {
		if err := insertJob(dbpool, job); err != nil {
			t.Fatalf("failed to insert job: %v", err)
		}
	}

	t.Cleanup(func() {
		for _, job := range jobs {
			if job.ID != 0 {
				_, err := dbpool.Exec(
					context.Background(),
					"DELETE FROM jobs WHERE id = $1",
					job.ID,
				)
				if err != nil {
					t.Errorf("failed to clean up job %d: %v", job.ID, err)
				}
			}
		}
	})

	claimedJob, err := claimNextJob(dbpool)
	if err != nil {
		t.Fatalf("claimNextJob returned an error: %v", err)
	}

	if claimedJob.ID != jobs[1].ID {
		t.Errorf("expected job ID %d, got %d",
			jobs[1].ID, claimedJob.ID)
	}

	if claimedJob.Status != JobStatusAssigned {
		t.Errorf("expected status ASSIGNED, got %s",
			claimedJob.Status)
	}
}
func TestConcurrentClaimNextJob(t *testing.T) {
    dbpool := setupTestDB(t)

    jobs := []*Job{
        {Name: "Concurrent Job A", Priority: 5, Status: JobStatusQueued},
        {Name: "Concurrent Job B", Priority: 3, Status: JobStatusQueued},
    }

    // Clean up only jobs created by this test.
    t.Cleanup(func() {
        for _, job := range jobs {
            if job.ID == 0 {
                continue
            }

            _, err := dbpool.Exec(
                context.Background(),
                "DELETE FROM jobs WHERE id = $1",
                job.ID,
            )
            if err != nil {
                t.Errorf("cleanup failed for job %d: %v", job.ID, err)
            }
        }
    })

    // Insert both jobs.
    for _, job := range jobs {
        if err := insertJob(dbpool, job); err != nil {
            t.Fatalf("failed to insert job: %v", err)
        }
    }

    type claimResult struct {
        job *Job
        err error
    }

    results := make(chan claimResult, 2)
    start := make(chan struct{})

    // Simulate two schedulers.
    for i := 0; i < 2; i++ {
        go func() {
            <-start

            job, err := claimNextJob(dbpool)
            results <- claimResult{
                job: job,
                err: err,
            }
        }()
    }

    // Release both goroutines at approximately the same time.
    close(start)

    first := <-results
    second := <-results

    if first.err != nil {
        t.Fatalf("first scheduler failed: %v", first.err)
    }

    if second.err != nil {
        t.Fatalf("second scheduler failed: %v", second.err)
    }

    if first.job.ID == second.job.ID {
        t.Fatalf(
            "both schedulers claimed the same job: ID %d",
            first.job.ID,
        )
    }

    if first.job.Status != JobStatusAssigned ||
        second.job.Status != JobStatusAssigned {
        t.Errorf("expected both jobs to be ASSIGNED")
    }
}
func TestClaimNextJobSkipsLockedJob(t *testing.T) {
    dbpool := setupTestDB(t)
    ctx := context.Background()

    jobs := []*Job{
        {Name: "Locked High Priority", Priority: 5, Status: JobStatusQueued},
        {Name: "Available Lower Priority", Priority: 3, Status: JobStatusQueued},
    }

    // Insert test jobs.
    for _, job := range jobs {
        if err := insertJob(dbpool, job); err != nil {
            t.Fatalf("failed to insert job: %v", err)
        }
    }

    // Clean up only this test's jobs.
    t.Cleanup(func() {
        for _, job := range jobs {
            if job.ID == 0 {
                continue
            }

            _, err := dbpool.Exec(
                ctx,
                "DELETE FROM jobs WHERE id = $1",
                job.ID,
            )
            if err != nil {
                t.Errorf("cleanup failed for job %d: %v", job.ID, err)
            }
        }
    })

    // Transaction A locks the highest-priority job.
    tx, err := dbpool.Begin(ctx)
    if err != nil {
        t.Fatalf("failed to begin transaction: %v", err)
    }
    defer tx.Rollback(ctx)

    var lockedJobID int

    err = tx.QueryRow(ctx, `
        SELECT id
        FROM jobs
        WHERE status = $1
        ORDER BY priority DESC, id ASC
        LIMIT 1
        FOR UPDATE
    `, JobStatusQueued).Scan(&lockedJobID)

    if err != nil {
        t.Fatalf("failed to lock job: %v", err)
    }

    if lockedJobID != jobs[0].ID {
        t.Fatalf("expected to lock job %d, got %d",
            jobs[0].ID, lockedJobID)
    }

    // Scheduler B should skip the locked job.
    claimedJob, err := claimNextJob(dbpool)
    if err != nil {
        t.Fatalf("failed to claim next available job: %v", err)
    }

    if claimedJob.ID != jobs[1].ID {
        t.Errorf("expected job %d, got job %d",
            jobs[1].ID, claimedJob.ID)
    }

    if claimedJob.Status != JobStatusAssigned {
        t.Errorf("expected ASSIGNED, got %s", claimedJob.Status)
    }

    // Release Transaction A's lock.
    if err := tx.Commit(ctx); err != nil {
        t.Fatalf("failed to commit transaction: %v", err)
    }

    // Verify Job A was not assigned.
    lockedJob, err := getJobByID(dbpool, jobs[0].ID)
    if err != nil {
        t.Fatalf("failed to retrieve locked job: %v", err)
    }

    if lockedJob.Status != JobStatusQueued {
        t.Errorf("expected locked job to remain QUEUED, got %s",
            lockedJob.Status)
    }
}
func TestRunScheduler(t *testing.T) {
    dbpool := setupTestDB(t)

    job := &Job{
        Name:     "Scheduler Test",
        Priority: 3,
        Status:   JobStatusQueued,
    }

    if err := insertJob(dbpool, job); err != nil {
        t.Fatalf("failed to insert job: %v", err)
    }

    t.Cleanup(func() {
        _, err := dbpool.Exec(
            context.Background(),
            "DELETE FROM jobs WHERE id = $1",
            job.ID,
        )
        if err != nil {
            t.Errorf("failed to clean up job: %v", err)
        }
    })

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    done := make(chan struct{})

    go func() {
        defer close(done)
        runScheduler(ctx, dbpool, time.Hour)
    }()

    deadline := time.After(2 * time.Second)
    ticker := time.NewTicker(10 * time.Millisecond)
    defer ticker.Stop()

    for {
        select {
        case <-deadline:
            t.Fatal("scheduler did not claim the job")

        case <-ticker.C:
            updatedJob, err := getJobByID(dbpool, job.ID)
            if err != nil {
                t.Fatalf("failed to retrieve job: %v", err)
            }

            if updatedJob.Status == JobStatusAssigned {
                cancel()

                select {
                case <-done:
                    return
                case <-time.After(time.Second):
                    t.Fatal("scheduler did not stop after cancellation")
                }
            }
        }
    }
}
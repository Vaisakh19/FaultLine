package main

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"os"
	"testing"

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

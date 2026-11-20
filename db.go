package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"os"
)

func connectDB() (*pgxpool.Pool, error) {
	connStr := os.Getenv("DATABASE_URL")

	dbpool, err := pgxpool.New(context.Background(), connStr)

	if err != nil {
		return nil, err
	}

	err = dbpool.Ping(context.Background())

	if err != nil {
		dbpool.Close()
		return nil, err
	}

	fmt.Println("Connected to PostgreSQL!")

	return dbpool, nil
}
func insertJob(dbpool *pgxpool.Pool, job *Job) error {
	query := `
    INSERT INTO jobs (name, priority, status)
    VALUES ($1, $2, $3)
    RETURNING id
`

	err := dbpool.QueryRow(
		context.Background(),
		query,
		job.Name,
		job.Priority,
		job.Status,
	).Scan(&job.ID)
	return err
}
func getJobs(dbpool *pgxpool.Pool) ([]Job, error) {
	query := `
        SELECT id, name, priority, status
        FROM jobs
        ORDER BY id
    `

	rows, err := dbpool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var jobs []Job

	for rows.Next() {
		var job Job

		err := rows.Scan(
			&job.ID,
			&job.Name,
			&job.Priority,
			&job.Status,
		)

		if err != nil {
			return nil, err
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}
func getJobByID(dbpool *pgxpool.Pool, id int) (Job, error) {
	query := `
        SELECT id, name, priority, status
        FROM jobs
        WHERE id = $1
    `

	var job Job

	err := dbpool.QueryRow(
		context.Background(),
		query,
		id,
	).Scan(
		&job.ID,
		&job.Name,
		&job.Priority,
		&job.Status,
	)

	if err != nil {
		return Job{}, err
	}

	return job, nil
}
func updateJobStatus(
	dbpool *pgxpool.Pool,
	id int,
	next JobStatus,
) error {
	ctx := context.Background()

	tx, err := dbpool.Begin(ctx)
	if err != nil {
		return err
	}

	defer tx.Rollback(ctx)

	var current JobStatus

	err = tx.QueryRow(
		ctx,
		`SELECT status FROM jobs WHERE id = $1 FOR UPDATE`,
		id,
	).Scan(&current)

	if errors.Is(err, pgx.ErrNoRows) {
		return ErrJobNotFound
	}

	if err != nil {
		return err
	}

	if !canTransition(current, next) {
		return ErrInvalidTransition
	}

	_, err = tx.Exec(
		ctx,
		`UPDATE jobs SET status = $1 WHERE id = $2`,
		next,
		id,
	)

	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}
func getNextJob(dbpool *pgxpool.Pool) (*Job, error) {
	query := `
        SELECT id, name, priority, status
        FROM jobs
        WHERE status = $1
        ORDER BY priority DESC, id ASC
        LIMIT 1
    `

	var job Job

	err := dbpool.QueryRow(
		context.Background(),
		query,
		JobStatusQueued,
	).Scan(
		&job.ID,
		&job.Name,
		&job.Priority,
		&job.Status,
	)

	if err != nil {
		return nil, err
	}

	return &job, nil
}
func claimNextJob(dbpool *pgxpool.Pool) (*Job, error) {
	ctx := context.Background()

	tx, err := dbpool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	query := `
        SELECT id, name, priority, status
        FROM jobs
        WHERE status = $1
        ORDER BY priority DESC, id ASC
        LIMIT 1
        FOR UPDATE SKIP LOCKED
    `

	var job Job

	err = tx.QueryRow(ctx, query, JobStatusQueued).Scan(
		&job.ID,
		&job.Name,
		&job.Priority,
		&job.Status,
	)

	if err != nil {
		return nil, err
	}

	// We'll update the job status here next.

	updateQuery := `
    UPDATE jobs
    SET status = $1
    WHERE id = $2
    RETURNING status
`

	err = tx.QueryRow(
		ctx,
		updateQuery,
		JobStatusAssigned,
		job.ID,
	).Scan(&job.Status)

	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return &job, nil
}

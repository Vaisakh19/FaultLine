package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
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
        INSERT INTO jobs (name, priority)
        VALUES ($1, $2)
        RETURNING id
    `

	err := dbpool.QueryRow(
		context.Background(),
		query,
		job.Name,
		job.Priority,
	).Scan(&job.ID)

	return err
}
func getJobs(dbpool *pgxpool.Pool) ([]Job, error) {
	query := `SELECT id, name, priority FROM jobs`

	rows, err := dbpool.Query(context.Background(), query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var jobs []Job

	for rows.Next() {
		var job Job

		err := rows.Scan(&job.ID, &job.Name, &job.Priority)
		if err != nil {
			return nil, err
		}

		jobs = append(jobs, job)
	}

	return jobs, rows.Err()
}
func getJobByID(dbpool *pgxpool.Pool, id int) (Job, error) {
    query := `
        SELECT id, name, priority
        FROM jobs
        WHERE id = $1
    `

    var job Job

    err := dbpool.QueryRow(
        context.Background(),
        query,
        id,
    ).Scan(&job.ID, &job.Name, &job.Priority)

    if err != nil {
        return Job{}, err
    }

    return job, nil
}
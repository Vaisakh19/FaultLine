package main

import (
	"encoding/json"
	"fmt"
    "context"
    "time"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"strings"
	"sync"
	"strconv"
	"github.com/jackc/pgx/v5"
	"errors"
    
)
type JobStatus string

const (
    JobStatusQueued    JobStatus = "QUEUED"
    JobStatusAssigned  JobStatus = "ASSIGNED"
    JobStatusRunning   JobStatus = "RUNNING"
    JobStatusCompleted JobStatus = "COMPLETED"
    JobStatusFailed    JobStatus = "FAILED"
)
type Job struct {
    ID       int    `json:"id"`
    Name     string `json:"name"`
    Priority int    `json:"priority"`
    Status   JobStatus `json:"status"`
}

var nextJobID = 1
var mu sync.Mutex

func createHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.Header().Set("Allow", http.MethodPost)
			http.Error(w, "Method not Allowed", http.StatusMethodNotAllowed)
			return
		}

		var job Job
		err := json.NewDecoder(r.Body).Decode(&job)
		if err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		if strings.TrimSpace(job.Name) == "" {
			http.Error(w, "Job cannot be empty", http.StatusBadRequest)
			return
		}
		if job.Priority < 1 || job.Priority > 5 {
			http.Error(w, "Priority must be between 1 and 5", http.StatusBadRequest)
			return
		}
		job.Status=JobStatusQueued
		err = insertJob(dbpool, &job)

		if err != nil {
			http.Error(w, "Failed to create job", http.StatusInternalServerError)
			return
		}
		fmt.Println("Job Received:", job.Name)
		fmt.Println("Job priority", job.Priority)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(job)
	}
}
func getJobsHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			w.Header().Set("Allow", http.MethodGet)
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		jobs, err := getJobs(dbpool)

		if err != nil {
			http.Error(w, "Failed to retrieve jobs", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jobs)
	}
}
func getJobByIDHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {

        id, err := strconv.Atoi(r.PathValue("id"))

        if err != nil {
            http.Error(w, "Invalid job ID", http.StatusBadRequest)
            return
        }
        job, err := getJobByID(dbpool, id)

        if err != nil {
            if err == pgx.ErrNoRows {
                http.Error(w, "Job not found", http.StatusNotFound)
                return
            }

            http.Error(w, "Failed to retrieve job", http.StatusInternalServerError)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(job)
    }
}
func patchJobStatusHandler(dbpool *pgxpool.Pool) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {

        // Extract job ID from URL
        id, err := strconv.Atoi(r.PathValue("id"))
        if err != nil || id <= 0 {
            http.Error(w, "Invalid job ID", http.StatusBadRequest)
            return
        }

        // Decode request body
        var request struct {
            Status JobStatus `json:"status"`
        }

        err = json.NewDecoder(r.Body).Decode(&request)
        if err != nil {
            http.Error(w, "Invalid JSON", http.StatusBadRequest)
            return
        }

        // Update job status
        err = updateJobStatus(dbpool, id, request.Status)

        if errors.Is(err, ErrJobNotFound) {
            http.Error(w, "Job not found", http.StatusNotFound)
            return
        }

        if errors.Is(err, ErrInvalidTransition) {
            http.Error(w, "Invalid status transition", http.StatusBadRequest)
            return
        }

        if err != nil {
            http.Error(w, "Failed to update job status", http.StatusInternalServerError)
            return
        }

        // Retrieve updated job
        job, err := getJobByID(dbpool, id)
        if err != nil {
            http.Error(w, "Failed to retrieve updated job", http.StatusInternalServerError)
            return
        }

        // Return updated job
        w.Header().Set("Content-Type", "application/json")
        json.NewEncoder(w).Encode(job)
    }
}
func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Request received:", r.Method)
	fmt.Fprintln(w, "Welcome to FaultLine")

}
func main() {
	dbpool, err := connectDB()
	if err != nil {
		fmt.Println("Database connection failed:", err)
		return
	}
	defer dbpool.Close()

	mux := http.NewServeMux()

	mux.HandleFunc("GET /", homeHandler)
	mux.HandleFunc("POST /jobs", createHandler(dbpool))
	mux.HandleFunc("GET /jobs", getJobsHandler(dbpool))
	mux.HandleFunc("GET /jobs/{id}", getJobByIDHandler(dbpool))
	mux.HandleFunc("PATCH /jobs/{id}/status", patchJobStatusHandler(dbpool))

	// Create a shared context for the scheduler.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Configure the HTTP server.
	server := &http.Server{
		Addr:    "127.0.0.1:8080",
		Handler: mux,
	}

	// Start the scheduler in a separate goroutine.
	schedulerErr := make(chan error, 1)

	go func() {
		schedulerErr <- runScheduler(ctx, dbpool, time.Second)
	}()

	// Start the HTTP server in a separate goroutine.
	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.ListenAndServe()
	}()

	fmt.Println("Faultline server running on port 8080")

	// Monitor both components.
	select {
	case err := <-schedulerErr:
		if err != nil {
			fmt.Println("Scheduler error:", err)
		}
		server.Close()

	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			fmt.Println("Server error:", err)
		}
		cancel()
	}
}

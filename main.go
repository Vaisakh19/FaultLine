package main

import (
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"strings"
	"sync"
	"strconv"
	"github.com/jackc/pgx/v5"
)

type Job struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Priority int    `json:"priority"`
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

    fmt.Println("Faultline server running on port 8080")

    err = http.ListenAndServe("127.0.0.1:8080", mux)
    if err != nil {
        fmt.Println("Server error:", err)
    }
}
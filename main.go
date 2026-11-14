package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
)

type Job struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Priority int    `json:"priority"`
}

var nextJobID = 1
var mu sync.Mutex

func createHandler(w http.ResponseWriter, r *http.Request) {
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
	mu.Lock()
	job.ID = nextJobID
	nextJobID++
	mu.Unlock()
	fmt.Println("Job Received:", job.Name)
	fmt.Println("Job priority", job.Priority)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
	//fmt.Fprintln(w, "Job received successfully")

}
func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Request received:", r.Method)
	fmt.Fprintln(w, "Welcome to FaultLine")

}
func main() {
	http.HandleFunc("/", homeHandler)
	fmt.Println("Port running on 8080")
	http.HandleFunc("/jobs", createHandler)
	// fmt.Println("Job received successfully")
	err := http.ListenAndServe("127.0.0.1:8080", nil)
	if err != nil {
		fmt.Println("Server error:", err)
	}

}

package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestCreateHandlerRejectsGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	recorder := httptest.NewRecorder()

	createHandler(nil)(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", recorder.Code)
	}
}
func TestGetJobsHandlerSuccess(t *testing.T) {
	dbpool := setupTestDB(t)

	job := Job{
		Name:     "GET jobs test",
		Priority: 3,
		Status:   JobStatusQueued,
	}

	err := insertJob(dbpool, &job)
	if err != nil {
		t.Fatalf("failed to insert test job: %v", err)
	}
	t.Cleanup(func() {
		_, err := dbpool.Exec(
			context.Background(),
			"DELETE FROM jobs WHERE id = $1",
			job.ID,
		)
		if err != nil {
			t.Errorf("failed to clean up test job: %v", err)
		}
	})

	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	recorder := httptest.NewRecorder()

	getJobsHandler(dbpool)(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	var jobs []Job
	err = json.NewDecoder(recorder.Body).Decode(&jobs)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	found := false
	for _, returnedJob := range jobs {
		if returnedJob.ID == job.ID {
			found = true
			break
		}
	}

	if !found {
		t.Error("inserted job was not found in GET response")
	}
}
func TestCreateHandlerRejectsInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"name":"Email Report","priority":}`))
	recorder := httptest.NewRecorder()
	createHandler(nil)(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", recorder.Code)
	}
}
func TestGetJobByIDHandlerSuccess(t *testing.T) {
	dbpool := setupTestDB(t)

	// Insert a test job
	job := Job{
		Name:     "Get job by ID test",
		Priority: 4,
		Status:   JobStatusQueued,
	}

	err := insertJob(dbpool, &job)
	if err != nil {
		t.Fatalf("failed to insert test job: %v", err)
	}

	// Clean up after the test
	t.Cleanup(func() {
		_, err := dbpool.Exec(
			context.Background(),
			"DELETE FROM jobs WHERE id = $1",
			job.ID,
		)
		if err != nil {
			t.Errorf("failed to clean up test job: %v", err)
		}
	})

	// Create request
	req := httptest.NewRequest(
		http.MethodGet,
		"/jobs/"+strconv.Itoa(job.ID),
		nil,
	)

	// Set the path parameter because we're calling the handler directly
	req.SetPathValue("id", strconv.Itoa(job.ID))

	recorder := httptest.NewRecorder()

	// Call handler
	getJobByIDHandler(dbpool)(recorder, req)

	// Check status
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}

	// Decode response
	var returnedJob Job
	err = json.NewDecoder(recorder.Body).Decode(&returnedJob)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// Verify returned job
	if returnedJob.ID != job.ID {
		t.Errorf("expected ID %d, got %d", job.ID, returnedJob.ID)
	}

	if returnedJob.Name != job.Name {
		t.Errorf("expected name %q, got %q", job.Name, returnedJob.Name)
	}
}
func TestCreateHandlerRejectsEmptyName(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"name":" ","priority":4}`))
	recorder := httptest.NewRecorder()
	createHandler(nil)(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", recorder.Code)
	}
}

func TestCreateHandlerRejectsInvalidPriority(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"name":"Data","priority":0}`))
	recorder := httptest.NewRecorder()
	createHandler(nil)(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", recorder.Code)
	}
}
func TestCreateHandlerSuccess(t *testing.T) {
	req := httptest.NewRequest(
		http.MethodPost,
		"/jobs",
		strings.NewReader(`{"name": "Email Report", "priority": 2}`),
	)

	recorder := httptest.NewRecorder()

	dbpool := setupTestDB(t)
	defer dbpool.Close()

	createHandler(dbpool)(recorder, req)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d", recorder.Code)
	}
	contentType := recorder.Header().Get("Content-Type")

	if contentType != "application/json" {
		t.Errorf("expected application/json, got %s", contentType)
	}
	var job Job
	err := json.NewDecoder(recorder.Body).Decode(&job)

	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if job.ID == 0 {
		t.Error("expected a non-zero job ID")
	}

	if job.Name != "Email Report" || job.Priority != 2 {
		t.Errorf("unexpected job data: %+v", job)
	}
}

func TestGetJobByIDNotFound(t *testing.T) {
	dbpool := setupTestDB(t)

	// Request a job ID that cannot exist
	req := httptest.NewRequest(
		http.MethodGet,
		"/jobs/-1",
		nil,
	)

	req.SetPathValue("id", "-1")

	recorder := httptest.NewRecorder()

	// Call handler
	getJobByIDHandler(dbpool)(recorder, req)

	// Verify status code
	if recorder.Code != http.StatusNotFound {
		t.Errorf(
			"expected status 404, got %d",
			recorder.Code,
		)
	}
}
func TestGetJobByIDInvalidID(t *testing.T) {
	dbpool := setupTestDB(t)

	req := httptest.NewRequest(
		http.MethodGet,
		"/jobs/abc",
		nil,
	)

	req.SetPathValue("id", "abc")

	recorder := httptest.NewRecorder()

	getJobByIDHandler(dbpool)(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf(
			"expected status 400, got %d",
			recorder.Code,
		)
	}
}
func TestPatchJobStatusSuccess(t *testing.T) {
	dbpool := setupTestDB(t)

	job := Job{
		Name:     "PATCH success test",
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

	body := `{"status":"ASSIGNED"}`

	req := httptest.NewRequest(
		http.MethodPatch,
		"/jobs/"+strconv.Itoa(job.ID)+"/status",
		strings.NewReader(body),
	)

	req.SetPathValue("id", strconv.Itoa(job.ID))

	recorder := httptest.NewRecorder()

	patchJobStatusHandler(dbpool)(recorder, req)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recorder.Code)
	}

	var updatedJob Job

	err = json.NewDecoder(recorder.Body).Decode(&updatedJob)
	if err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if updatedJob.Status != JobStatusAssigned {
		t.Errorf(
			"expected ASSIGNED, got %s",
			updatedJob.Status,
		)
	}
}
func TestPatchJobStatusInvalidTransition(t *testing.T) {
	dbpool := setupTestDB(t)

	job := Job{
		Name:     "PATCH invalid transition test",
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

	body := `{"status":"COMPLETED"}`

	req := httptest.NewRequest(
		http.MethodPatch,
		"/jobs/"+strconv.Itoa(job.ID)+"/status",
		strings.NewReader(body),
	)

	req.SetPathValue("id", strconv.Itoa(job.ID))

	recorder := httptest.NewRecorder()

	patchJobStatusHandler(dbpool)(recorder, req)

	// Verify HTTP response
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf(
			"expected 400, got %d",
			recorder.Code,
		)
	}

	// Verify database was not modified
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
func TestPatchJobStatusNotFound(t *testing.T) {
	dbpool := setupTestDB(t)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/jobs/-1/status",
		strings.NewReader(`{"status":"RUNNING"}`),
	)

	req.SetPathValue("id", "99999")

	recorder := httptest.NewRecorder()

	patchJobStatusHandler(dbpool)(recorder, req)

	if recorder.Code != http.StatusNotFound {
		t.Errorf(
			"expected 404, got %d",
			recorder.Code,
		)
	}
}
func TestPatchJobStatusInvalidID(t *testing.T) {
	dbpool := setupTestDB(t)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/jobs/abc/status",
		strings.NewReader(`{"status":"RUNNING"}`),
	)

	req.SetPathValue("id", "abc")

	recorder := httptest.NewRecorder()

	patchJobStatusHandler(dbpool)(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf(
			"expected 400, got %d",
			recorder.Code,
		)
	}
}
func TestPatchJobStatusInvalidJSON(t *testing.T) {
	dbpool := setupTestDB(t)

	req := httptest.NewRequest(
		http.MethodPatch,
		"/jobs/1/status",
		strings.NewReader(`{"status":}`),
	)

	req.SetPathValue("id", "1")

	recorder := httptest.NewRecorder()

	patchJobStatusHandler(dbpool)(recorder, req)

	if recorder.Code != http.StatusBadRequest {
		t.Errorf(
			"expected 400, got %d",
			recorder.Code,
		)
	}
}

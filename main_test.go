package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestCreateHandlerRejectsGet(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/jobs", nil)
	recorder := httptest.NewRecorder()

	createHandler(recorder, req)

	if recorder.Code != http.StatusMethodNotAllowed {
		t.Errorf("expected status 405, got %d", recorder.Code)
	}
}

func TestCreateHandlerRejectsInvalidJSON(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"name":"Email Report","priority":}`))
	recorder := httptest.NewRecorder()
	createHandler(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", recorder.Code)
	}
}

func TestCreateHandlerRejectsEmptyName(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"name":" ","priority":4}`))
	recorder := httptest.NewRecorder()
	createHandler(recorder, req)
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", recorder.Code)
	}
}

func TestCreateHandlerRejectsInvalidPriority(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/jobs", strings.NewReader(`{"name":"Data","priority":0}`))
	recorder := httptest.NewRecorder()
	createHandler(recorder, req)
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

	createHandler(recorder, req)

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

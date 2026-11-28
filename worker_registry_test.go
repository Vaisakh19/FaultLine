package main

import "testing"

func TestRegisterWorker(t *testing.T) {
	registry := NewWorkerRegistry()

	worker := &Worker{
		ID:       "worker-01",
		Capacity: 4,
	}

	registry.Register(worker)

	got, exists := registry.Get("worker-01")

	if !exists {
		t.Fatal("expected worker to be registered")
	}

	if got.ID != worker.ID {
		t.Errorf("expected worker ID %s, got %s", worker.ID, got.ID)
	}
}

func TestGetNonexistentWorker(t *testing.T) {
	registry := NewWorkerRegistry()

	_, exists := registry.Get("worker-99")

	if exists {
		t.Error("expected worker not to exist")
	}
}

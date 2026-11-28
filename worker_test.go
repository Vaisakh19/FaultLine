package main

import (
    "testing"
    "errors"
)
func TestAvailableSlots(t *testing.T) {
    worker := Worker{
        ID:         "worker-01",
        Capacity:   4,
        ActiveJobs: 3,
    }

    got := worker.AvailableSlots()

    if got != 1 {
        t.Errorf("expected 1 available slot, got %d", got)
    }
}
func TestTryAcquireSlot(t *testing.T) {
    worker := &Worker{
        Capacity: 2,
    }

    if err := worker.TryAcquireSlot(); err != nil {
        t.Fatal(err)
    }

    if worker.AvailableSlots() != 1 {
        t.Errorf("expected 1 slot, got %d", worker.AvailableSlots())
    }
}

func TestWorkerAtCapacity(t *testing.T) {
    worker := &Worker{
        Capacity:   1,
        ActiveJobs: 1,
    }

    err := worker.TryAcquireSlot()

    if !errors.Is(err, ErrWorkerAtCapacity) {
        t.Errorf("expected ErrWorkerAtCapacity, got %v", err)
    }
}

func TestReleaseSlot(t *testing.T) {
    worker := &Worker{
        Capacity:   2,
        ActiveJobs: 1,
    }

    if err := worker.ReleaseSlot(); err != nil {
        t.Fatal(err)
    }

    if worker.AvailableSlots() != 2 {
        t.Errorf("expected 2 slots, got %d", worker.AvailableSlots())
    }
}
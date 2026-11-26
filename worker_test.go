package main

import "testing"

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
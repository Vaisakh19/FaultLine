package main

import (
    "errors"
    "sync"
)

var (
    ErrWorkerAtCapacity = errors.New("worker is at full capacity")
    ErrNoActiveJobs     = errors.New("worker has no active jobs")
)

type Worker struct {
    mu         sync.Mutex
    ID         string
    Capacity   int
    ActiveJobs int
}

func (w *Worker) AvailableSlots() int {
    w.mu.Lock()
    defer w.mu.Unlock()

    return w.Capacity - w.ActiveJobs
}

func (w *Worker) TryAcquireSlot() error {
    w.mu.Lock()
    defer w.mu.Unlock()

    if w.ActiveJobs >= w.Capacity {
        return ErrWorkerAtCapacity
    }

    w.ActiveJobs++
    return nil
}

func (w *Worker) ReleaseSlot() error {
    w.mu.Lock()
    defer w.mu.Unlock()

    if w.ActiveJobs <= 0 {
        return ErrNoActiveJobs
    }

    w.ActiveJobs--
    return nil
}
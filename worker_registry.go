package main

import "errors"

var (
	ErrNilWorker       = errors.New("worker cannot be nil")
	ErrEmptyWorkerID   = errors.New("worker ID cannot be empty")
	ErrDuplicateWorker = errors.New("worker already registered")
)

type WorkerRegistry struct {
	workers map[string]*Worker
}

func NewWorkerRegistry() *WorkerRegistry {
	return &WorkerRegistry{
		workers: make(map[string]*Worker),
	}
}

func (r *WorkerRegistry) Register(worker *Worker) error {
	if worker == nil {
		return ErrNilWorker
	}

	if worker.ID == "" {
		return ErrEmptyWorkerID
	}

	if _, exists := r.workers[worker.ID]; exists {
		return ErrDuplicateWorker
	}

	r.workers[worker.ID] = worker
	return nil
}

func (r *WorkerRegistry) Get(id string) (*Worker, bool) {
	worker, exists := r.workers[id]
	return worker, exists
}

package main

import ( 
	"errors"
	"sync"
)

var (
	ErrNilWorker       = errors.New("worker cannot be nil")
	ErrEmptyWorkerID   = errors.New("worker ID cannot be empty")
	ErrDuplicateWorker = errors.New("worker already registered")
	ErrWorkerNotFound  = errors.New("worker not found")
)

type WorkerRegistry struct {
	mu      sync.Mutex
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
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.workers[worker.ID]; exists {
		return ErrDuplicateWorker
	}

	r.workers[worker.ID] = worker
	return nil
}

func (r *WorkerRegistry) Get(id string) (*Worker, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	worker, exists := r.workers[id]
	return worker, exists
}

func (r *WorkerRegistry) Unregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, exists := r.workers[id]; !exists {
		return ErrWorkerNotFound
	}

	delete(r.workers, id)
	return nil
}

package main

type Worker struct {
    ID         string
    Capacity   int
    ActiveJobs int
}

func (w Worker) AvailableSlots() int {
    return w.Capacity - w.ActiveJobs
}
package main
import "errors"

var (
    ErrJobNotFound      = errors.New("job not found")
    ErrInvalidTransition = errors.New("invalid job status transition")
)
func canTransition(current, next JobStatus) bool {
    switch current {
    case JobStatusQueued:
        return next == JobStatusAssigned

    case JobStatusAssigned:
        return next == JobStatusRunning

    case JobStatusRunning:
        return next == JobStatusCompleted ||
            next == JobStatusFailed

    default:
        return false
    }
}
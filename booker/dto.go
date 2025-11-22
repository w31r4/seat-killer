package booker

import (
	"net/http"
	"time"
)

// BookingRequest encapsulates all parameters needed for a seat booking attempt.
type BookingRequest struct {
	Client    *http.Client
	UserID    string
	SeatID    int
	BeginTime time.Time
	Duration  time.Duration
}

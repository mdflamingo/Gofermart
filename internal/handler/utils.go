package handler

import (
	"net/http"
	"sync"
	"time"
)

var (
	requestCounts = make(map[string][]time.Time)
	mu            sync.RWMutex
	limit         = 10
	window        = time.Minute
)

func rateLimit(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := r.RemoteAddr

		mu.Lock()
		defer mu.Unlock()

		now := time.Now()

		var valid []time.Time
		for _, t := range requestCounts[ip] {
			if now.Sub(t) <= window {
				valid = append(valid, t)
			}
		}

		if len(valid) >= limit {
			w.Header().Set("Content-Type", "text/plain")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			w.Write([]byte("No more than N requests per minute allowed"))
			return
		}

		requestCounts[ip] = append(valid, now)

		next.ServeHTTP(w, r)
	}
}

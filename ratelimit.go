package circularbuffer

import (
	"context"
	"math"
	"sync"
	"time"
)

// RateLimiter is an interface which can be used to implement
// rate limiting.
type RateLimiter interface {
	// Allow returns true if call should be allowed, false in case
	// you should rate limit.
	//
	// Deprecated: In favour of AllowContext
	Allow(string) bool

	// AllowContext is like Allow but accepts an additional
	// context.Context, e.g. to support OpenTracing.
	AllowContext(context.Context, string) bool

	// Close cleans up the RateLimiter implementation.
	Close()
	Oldest(string) time.Time
	Delta(string) time.Duration
	Resize(string, int)
	// RetryAfter returns how many seconds until the next allowed request
	RetryAfter(string) int
}

// NewRateLimiter returns a new initialized RateLimitter with maxHits
// as the maximal number of hits per time.Duration d. This can be used
// to implement maximum number of requests for a backend to protect
// from a known scaling limit.
func NewRateLimiter(maxHits int, d time.Duration) RateLimiter {
	return NewCircularBuffer(maxHits, d)
}

// Allow returns true if there is a free bucket and we should not rate
// limit, if not it will return false, which means ratelimit.
//
// Deprecated: In favour of AllowContext
func (cb *CircularBuffer) Allow(s string) bool {
	return cb.Add(time.Now())
}

// Allow returns true if there is a free bucket and we should not rate
// limit, if not it will return false, which means ratelimit.
func (cb *CircularBuffer) AllowContext(ctx context.Context, s string) bool {
	return cb.Add(time.Now())
}

// Close implements the RateLimiter interface to shutdown, nothing to
// do.
func (*CircularBuffer) Close() {}

// Oldest implements the RateLimiter interface
func (cb *CircularBuffer) Oldest(string) time.Time {
	cb.RLock()
	cur := cb.slots[cb.offset]
	cb.RUnlock()
	return cur
}

// Current implements the RateLimiter interface
func (cb *CircularBuffer) Current(string) time.Time {
	return cb.current()
}

// Delta returns the diffence between the current and the oldest value in
// the buffer, i.e. maxHits / Delta() => rate
func (cb *CircularBuffer) Delta(string) time.Duration {
	return cb.delta()
}

// Resize resizes the circular buffer to the given size. Resizing to a size
// <= 0 is not performed
func (cb *CircularBuffer) Resize(_ string, n int) {
	cb.Lock()
	cb.resize(n)
	cb.Unlock()
}

// RetryAfter returns how many seconds one should wait until the next request
// is allowed.
func (cb *CircularBuffer) RetryAfter(string) int {
	retryAfter := cb.retryAfter()
	ms := retryAfter / time.Millisecond
	secs := math.Ceil(float64(ms) / 1000)
	return int(secs)
}

// ClientRateLimiter implements the RateLimiter interface and does
// rate limiting based on the the String passed to Allow(). This can
// be used to limit per client calls to the backend. For example you
// can slow down user enumeration or dictionary attacks to /login
// APIs.
type ClientRateLimiter struct {
	sync.RWMutex
	bag        map[string]*CircularBuffer
	maxHits    int
	timeWindow time.Duration
	quitCH     chan struct{}
}

// NewRateLimiter returns a new initialized RateLimitter with maxHits is
// the maximal number of hits per time.Duration d.
func NewClientRateLimiter(maxHits int, d, cleanInterval time.Duration) *ClientRateLimiter {
	quit := make(chan struct{})
	crl := &ClientRateLimiter{
		bag:        make(map[string]*CircularBuffer),
		maxHits:    maxHits,
		timeWindow: d,
		quitCH:     quit,
	}
	go crl.startCleanerDaemon(cleanInterval)
	return crl
}

// Allow tries to add s to a circularbuffer and returns true if we have
// a free bucket, if not it will return false, which means ratelimit.
//
// Deprecated: In favour of allow context
func (rl *ClientRateLimiter) Allow(s string) bool {
	var source *CircularBuffer
	var present bool

	rl.RLock()
	if source, present = rl.bag[s]; !present {
		rl.RUnlock()
		rl.Lock()
		source = NewCircularBuffer(rl.maxHits, rl.timeWindow)
		rl.bag[s] = source
		rl.Unlock()
	} else {
		rl.RUnlock()
	}
	present = source.Add(time.Now())
	return present
}

// AllowContext tries to add s to a circularbuffer and returns true if we have
// a free bucket, if not it will return false, which means ratelimit with an additional
// context.Context.
func (rl *ClientRateLimiter) AllowContext(ctx context.Context, s string) bool {
	var source *CircularBuffer
	var present bool

	rl.RLock()
	if source, present = rl.bag[s]; !present {
		rl.RUnlock()
		rl.Lock()
		source = NewCircularBuffer(rl.maxHits, rl.timeWindow)
		rl.bag[s] = source
		rl.Unlock()
	} else {
		rl.RUnlock()
	}
	present = source.Add(time.Now())
	return present
}

func (rl *ClientRateLimiter) Oldest(s string) time.Time {
	rl.RLock()
	if _, present := rl.bag[s]; !present {
		rl.RUnlock()
		return time.Time{}
	}
	delta := rl.bag[s].Oldest(s)
	rl.RUnlock()
	return delta
}

func (rl *ClientRateLimiter) Current(s string) time.Time {
	rl.RLock()
	if _, present := rl.bag[s]; !present {
		rl.RUnlock()
		return time.Time{}
	}
	delta := rl.bag[s].Current(s)
	rl.RUnlock()
	return delta
}

// Delta returns the diffence between the current and the oldest value in
// the buffer, i.e. maxHits / Delta() => rate
func (rl *ClientRateLimiter) Delta(s string) time.Duration {
	rl.RLock()
	if _, present := rl.bag[s]; !present {
		rl.RUnlock()
		return time.Duration(time.Hour * 24)
	}
	delta := rl.bag[s].delta()
	rl.RUnlock()
	return delta
}

// Resize resizes the given circular buffer to the given size. Resizing to a size
// <= 0 is not performed
func (rl *ClientRateLimiter) Resize(s string, n int) {
	rl.RLock()
	if _, present := rl.bag[s]; !present {
		rl.RUnlock()
		return
	}
	rl.RUnlock()
	rl.Lock()
	rl.bag[s].resize(n)
	rl.Unlock()
}

// RetryAfter returns how many seconds one should wait until the next request
// is allowed.
func (rl *ClientRateLimiter) RetryAfter(s string) int {
	rl.RLock()
	if _, present := rl.bag[s]; !present {
		rl.RUnlock()
		return 0
	}
	retryAfter := rl.bag[s].RetryAfter(s)
	rl.RUnlock()
	return retryAfter
}

// DeleteOld removes old entries from state bag
func (rl *ClientRateLimiter) DeleteOld() {
	rl.Lock()
	for k, cb := range rl.bag {
		if !cb.InUse() {
			delete(rl.bag, k)
		}
	}
	rl.Unlock()
}

// Close will stop the cleanup goroutine
func (rl *ClientRateLimiter) Close() {
	close(rl.quitCH)
}

func (rl *ClientRateLimiter) startCleanerDaemon(d time.Duration) {
	for {
		select {
		case <-rl.quitCH:
			return
		case <-time.After(d):
			rl.DeleteOld()
		}
	}
}

package sendpolicy

import (
	"errors"
	"sync"
	"time"
)

var ErrDailyLimit = errors.New("daily message limit reached")

type Policy struct {
	mu       sync.Mutex
	maxDaily int
	now      func() time.Time
	counts   map[string]counter
}

type counter struct {
	day   string
	count int
}

func New(maxDaily int) *Policy {
	return &Policy{maxDaily: maxDaily, now: time.Now, counts: make(map[string]counter)}
}

// Allow counts actual outbound messages, not AI turns.
func (p *Policy) Allow(accountID string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.maxDaily <= 0 {
		return ErrDailyLimit
	}
	day := p.now().UTC().Format("2006-01-02")
	c := p.counts[accountID]
	if c.day != day {
		c = counter{day: day}
	}
	if c.count >= p.maxDaily {
		return ErrDailyLimit
	}
	c.count++
	p.counts[accountID] = c
	return nil
}

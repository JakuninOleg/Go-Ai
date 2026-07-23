package observability

import (
	"strconv"
	"sync"
	"time"
)

type Metrics struct {
	mu sync.Mutex

	startTime time.Time

	requestsTotal          uint64
	chatRequestsTotal      uint64
	successTotal           uint64
	errorTotal             uint64
	authFailuresTotal      uint64
	fallbackTotal          uint64
	streamingRequestsTotal uint64

	providerRequests map[string]uint64
	statusCodes      map[int]uint64
	lastRequestAt    *time.Time
}

type Snapshot struct {
	Status                 string            `json:"status"`
	StartTime              time.Time         `json:"start_time"`
	UptimeSeconds          int64             `json:"uptime_seconds"`
	RequestsTotal          uint64            `json:"requests_total"`
	ChatRequestsTotal      uint64            `json:"chat_requests_total"`
	SuccessTotal           uint64            `json:"success_total"`
	ErrorTotal             uint64            `json:"error_total"`
	AuthFailuresTotal      uint64            `json:"auth_failures_total"`
	FallbackTotal          uint64            `json:"fallback_total"`
	StreamingRequestsTotal uint64            `json:"streaming_requests_total"`
	ProviderRequests       map[string]uint64 `json:"provider_requests"`
	StatusCodes            map[string]uint64 `json:"status_codes"`
	LastRequestAt          *time.Time        `json:"last_request_at,omitempty"`
}

func NewMetrics() *Metrics {
	return &Metrics{
		startTime:        time.Now().UTC(),
		providerRequests: make(map[string]uint64),
		statusCodes:      make(map[int]uint64),
	}
}

func (m *Metrics) RecordHTTP(status int) {
	if m == nil {
		return
	}
	if status == 0 {
		status = 200
	}

	now := time.Now().UTC()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.requestsTotal++
	m.statusCodes[status]++
	m.lastRequestAt = &now

	if status >= 400 {
		m.errorTotal++
		return
	}
	m.successTotal++
}

func (m *Metrics) RecordAuthFailure() {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.authFailuresTotal++
}

func (m *Metrics) RecordChatRequest(stream bool) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	m.chatRequestsTotal++
	if stream {
		m.streamingRequestsTotal++
	}
}

func (m *Metrics) RecordProviderRequest(provider string, fallbackUsed bool) {
	if m == nil {
		return
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if provider != "" {
		m.providerRequests[provider]++
	}
	if fallbackUsed {
		m.fallbackTotal++
	}
}

func (m *Metrics) Snapshot() Snapshot {
	if m == nil {
		return NewMetrics().Snapshot()
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	providerRequests := make(map[string]uint64, len(m.providerRequests))
	for provider, total := range m.providerRequests {
		providerRequests[provider] = total
	}

	statusCodes := make(map[string]uint64, len(m.statusCodes))
	for status, total := range m.statusCodes {
		statusCodes[strconv.Itoa(status)] = total
	}

	var lastRequestAt *time.Time
	if m.lastRequestAt != nil {
		last := *m.lastRequestAt
		lastRequestAt = &last
	}

	return Snapshot{
		Status:                 "ok",
		StartTime:              m.startTime,
		UptimeSeconds:          int64(time.Since(m.startTime).Seconds()),
		RequestsTotal:          m.requestsTotal,
		ChatRequestsTotal:      m.chatRequestsTotal,
		SuccessTotal:           m.successTotal,
		ErrorTotal:             m.errorTotal,
		AuthFailuresTotal:      m.authFailuresTotal,
		FallbackTotal:          m.fallbackTotal,
		StreamingRequestsTotal: m.streamingRequestsTotal,
		ProviderRequests:       providerRequests,
		StatusCodes:            statusCodes,
		LastRequestAt:          lastRequestAt,
	}
}

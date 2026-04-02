package middleware

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"log"
	nethttp "net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	sharedhttp "github.com/dipesh/bifrost/backend/internal/shared/http"
	"github.com/gin-gonic/gin"
)

const (
	defaultJSONBodyLimit  = 1 << 20
	authJSONBodyLimit     = 64 << 10
	agentJSONBodyLimit    = 16 << 10
	snapshotJSONBodyLimit = 4 << 20
	globalInFlightCap     = 128
	snapshotInFlightCap   = 16
	requestIDKey          = "request_id"
)

var requestCounter uint64

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := strings.TrimSpace(c.GetHeader("X-Request-ID"))
		if requestID == "" {
			requestID = newRequestID()
		}
		c.Set(requestIDKey, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func SecurityLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Next()

		status := c.Writer.Status()
		if status != nethttp.StatusUnauthorized &&
			status != nethttp.StatusForbidden &&
			status != nethttp.StatusTooManyRequests &&
			status != nethttp.StatusRequestEntityTooLarge {
			return
		}

		requestID, _ := c.Get(requestIDKey)
		log.Printf(
			"security_event request_id=%v status=%d method=%s path=%s client_ip=%s duration_ms=%d",
			requestID,
			status,
			c.Request.Method,
			c.Request.URL.Path,
			c.ClientIP(),
			time.Since(startedAt).Milliseconds(),
		)
	}
}

func BodyLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		limit := requestBodyLimit(c.Request.Method, c.Request.URL.Path)
		if limit <= 0 || c.Request.Body == nil {
			c.Next()
			return
		}

		c.Request.Body = nethttp.MaxBytesReader(c.Writer, c.Request.Body, limit)
		c.Next()
	}
}

func InFlightLimit() gin.HandlerFunc {
	global := make(chan struct{}, globalInFlightCap)
	snapshot := make(chan struct{}, snapshotInFlightCap)

	return func(c *gin.Context) {
		if !tryAcquire(global) {
			c.AbortWithStatusJSON(nethttp.StatusTooManyRequests, sharedhttp.Error("too many concurrent requests", "RATE_LIMITED", nil))
			return
		}
		defer release(global)

		if isSnapshotRoute(c.Request.URL.Path) {
			if !tryAcquire(snapshot) {
				c.AbortWithStatusJSON(nethttp.StatusTooManyRequests, sharedhttp.Error("too many concurrent snapshot requests", "RATE_LIMITED", nil))
				return
			}
			defer release(snapshot)
		}

		c.Next()
	}
}

type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
}

type tokenBucket struct {
	tokens     float64
	capacity   float64
	refillRate float64
	lastRefill time.Time
	lastSeen   time.Time
}

func NewPublicRateLimit() gin.HandlerFunc {
	limiter := &rateLimiter{buckets: map[string]*tokenBucket{}}
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method
		ip := clientKey(c.ClientIP())

		switch {
		case method == nethttp.MethodPost && path == "/api/v1/auth/login":
			email := extractJSONField(c, "email")
			if !enforceRate(c, limiter, "login:ip:"+ip, 20, time.Minute) {
				return
			}
			emailKey := strings.ToLower(strings.TrimSpace(email))
			if emailKey == "" {
				emailKey = "_"
			}
			if !enforceRate(c, limiter, "login:email:"+ip+":"+emailKey, 5, time.Minute) {
				return
			}
		case method == nethttp.MethodPost && path == "/api/v1/auth/bootstrap":
			if !enforceRate(c, limiter, "bootstrap:"+ip, 3, time.Minute) {
				return
			}
		case method == nethttp.MethodPost && path == "/api/v1/auth/invites/accept":
			if !enforceRate(c, limiter, "invite-accept:"+ip, 5, time.Minute) {
				return
			}
		case method == nethttp.MethodGet && strings.HasPrefix(path, "/api/v1/auth/invites/"):
			if !enforceRate(c, limiter, "invite-detail:"+ip, 30, time.Minute) {
				return
			}
		case method == nethttp.MethodPost && path == "/api/v1/agent/enroll":
			if !enforceRate(c, limiter, "agent-enroll-ip:"+ip, 10, time.Minute) {
				return
			}
			key := clientKey(strings.TrimSpace(c.GetHeader("X-Agent-Key")))
			if key != "_" && !enforceRate(c, limiter, "agent-enroll-key:"+key, 5, time.Minute) {
				return
			}
		case method == nethttp.MethodPost && path == "/api/v1/agent/heartbeat":
			key := clientKey(strings.TrimSpace(c.GetHeader("X-Agent-Key")))
			if !enforceRate(c, limiter, "agent-heartbeat:"+key, 12, time.Minute) {
				return
			}
		case method == nethttp.MethodPost && path == "/api/v1/agent/snapshot":
			key := clientKey(strings.TrimSpace(c.GetHeader("X-Agent-Key")))
			if !enforceRate(c, limiter, "agent-snapshot:"+key, 12, time.Minute) {
				return
			}
		case method == nethttp.MethodGet && (path == "/api/v1/agent/install" || path == "/api/v1/agent/install.sh"):
			if !enforceRate(c, limiter, "agent-install:"+ip, 60, time.Minute) {
				return
			}
		}

		c.Next()
	}
}

func NewAuthenticatedRateLimit() gin.HandlerFunc {
	limiter := &rateLimiter{buckets: map[string]*tokenBucket{}}
	return func(c *gin.Context) {
		userID := clientKey(c.GetString("user_id"))
		if isLogsOrEventsPath(c.Request.URL.Path) {
			if !enforceRate(c, limiter, "authed-logs:"+userID, 60, time.Minute) {
				return
			}
		} else {
			if !enforceRate(c, limiter, "authed:"+userID, 240, time.Minute) {
				return
			}
		}
		c.Next()
	}
}

func requestBodyLimit(method, path string) int64 {
	switch {
	case method == nethttp.MethodGet || method == nethttp.MethodDelete:
		return 0
	case isAuthMutationPath(path) || strings.HasPrefix(path, "/api/v1/admin/"):
		return authJSONBodyLimit
	case path == "/api/v1/agent/heartbeat" || path == "/api/v1/agent/enroll":
		return agentJSONBodyLimit
	case path == "/api/v1/agent/snapshot":
		return snapshotJSONBodyLimit
	default:
		return defaultJSONBodyLimit
	}
}

func isAuthMutationPath(path string) bool {
	switch path {
	case "/api/v1/auth/bootstrap", "/api/v1/auth/login", "/api/v1/auth/logout", "/api/v1/auth/me", "/api/v1/auth/me/password", "/api/v1/auth/invites/accept":
		return true
	default:
		return false
	}
}

func isSnapshotRoute(path string) bool {
	return path == "/api/v1/agent/snapshot"
}

func isLogsOrEventsPath(path string) bool {
	return strings.HasSuffix(path, "/logs") || strings.HasSuffix(path, "/events")
}

func newRequestID() string {
	value := atomic.AddUint64(&requestCounter, 1)
	return time.Now().UTC().Format("20060102150405.000000000") + "-" + strconv.FormatUint(value, 10)
}

func tryAcquire(ch chan struct{}) bool {
	select {
	case ch <- struct{}{}:
		return true
	default:
		return false
	}
}

func release(ch chan struct{}) {
	select {
	case <-ch:
	default:
	}
}

func clientKey(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "_"
	}
	return value
}

func extractJSONField(c *gin.Context, field string) string {
	if c.Request.Body == nil {
		return ""
	}

	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		var maxBytesErr *nethttp.MaxBytesError
		if errors.As(err, &maxBytesErr) {
			c.AbortWithStatusJSON(nethttp.StatusRequestEntityTooLarge, sharedhttp.Error("request body exceeds the allowed size", "REQUEST_TOO_LARGE", gin.H{
				"limit_bytes": maxBytesErr.Limit,
			}))
		} else {
			c.AbortWithStatusJSON(nethttp.StatusBadRequest, sharedhttp.Error("invalid request body", "INVALID_REQUEST", gin.H{"reason": err.Error()}))
		}
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(body))

	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	value, _ := payload[field].(string)
	return value
}

func enforceRate(c *gin.Context, limiter *rateLimiter, key string, capacity int, window time.Duration) bool {
	if limiter.allow(key, capacity, window) {
		return true
	}

	c.AbortWithStatusJSON(nethttp.StatusTooManyRequests, sharedhttp.Error("request rate limit exceeded", "RATE_LIMITED", nil))
	return false
}

func (l *rateLimiter) allow(key string, capacity int, window time.Duration) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if len(l.buckets) > 4096 {
		for bucketKey, bucket := range l.buckets {
			if now.Sub(bucket.lastSeen) > 10*time.Minute {
				delete(l.buckets, bucketKey)
			}
		}
	}

	bucket, ok := l.buckets[key]
	if !ok {
		bucket = &tokenBucket{
			tokens:     float64(capacity),
			capacity:   float64(capacity),
			refillRate: float64(capacity) / window.Seconds(),
			lastRefill: now,
			lastSeen:   now,
		}
		l.buckets[key] = bucket
	}

	elapsed := now.Sub(bucket.lastRefill).Seconds()
	if elapsed > 0 {
		bucket.tokens += elapsed * bucket.refillRate
		if bucket.tokens > bucket.capacity {
			bucket.tokens = bucket.capacity
		}
		bucket.lastRefill = now
	}
	bucket.lastSeen = now

	if bucket.tokens < 1 {
		return false
	}

	bucket.tokens--
	return true
}

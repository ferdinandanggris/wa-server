package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	keyPrefix    = "agent:"
	heartbeatTTL = 30 * time.Second
)

type Status string

const (
	StatusRunning Status = "running"
	StatusStopped Status = "stopped"
	StatusIdle    Status = "idle"
)

type AgentInfo struct {
	ID             string    `json:"id"`
	Status         Status    `json:"status"`
	LastHeartbeat  time.Time `json:"last_heartbeat"`
	StartedAt      time.Time `json:"started_at"`
	MessagesSent   int64     `json:"messages_sent"`
	MessagesFailed int64     `json:"messages_failed"`
}

type Tracker struct {
	rdb    *redis.Client
	key    string
	info   *AgentInfo
	mu     sync.Mutex
	stopCh chan struct{}
}

func NewTracker(rdb *redis.Client, agentID string) *Tracker {
	return &Tracker{
		rdb: rdb,
		key: keyPrefix + agentID,
		info: &AgentInfo{
			ID:        agentID,
			Status:    StatusIdle,
			StartedAt: time.Now(),
		},
		stopCh: make(chan struct{}),
	}
}

func (t *Tracker) Start(ctx context.Context) {
	t.mu.Lock()
	t.info.Status = StatusRunning
	t.mu.Unlock()

	go t.heartbeatLoop(ctx)
	slog.Info("agent tracker started", "agent_id", t.info.ID)
}

func (t *Tracker) Stop() {
	close(t.stopCh)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	t.mu.Lock()
	t.info.Status = StatusStopped
	_ = t.save(ctx)
	t.mu.Unlock()

	slog.Info("agent tracker stopped", "agent_id", t.info.ID)
}

func (t *Tracker) SetIdle() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.info.Status = StatusIdle
}

func (t *Tracker) SetRunning() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.info.Status = StatusRunning
}

func (t *Tracker) IncMessagesSent() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.info.MessagesSent++
}

func (t *Tracker) IncMessagesFailed() {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.info.MessagesFailed++
}

func (t *Tracker) heartbeatLoop(ctx context.Context) {
	ticker := time.NewTicker(heartbeatTTL / 2)
	defer ticker.Stop()

	for {
		select {
		case <-t.stopCh:
			return
		case <-ticker.C:
			t.mu.Lock()
			t.info.LastHeartbeat = time.Now()
			err := t.save(ctx)
			t.mu.Unlock()
			if err != nil {
				slog.Error("agent heartbeat failed", "agent_id", t.info.ID, "error", err)
			}
		}
	}
}

func (t *Tracker) save(ctx context.Context) error {
	data, err := json.Marshal(t.info)
	if err != nil {
		return fmt.Errorf("marshal agent info: %w", err)
	}
	return t.rdb.Set(ctx, t.key, data, heartbeatTTL).Err()
}

func NewRedisClient(addr, password string, db int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis ping: %w", err)
	}

	return rdb, nil
}

type AgentSnapshot struct {
	ID             string    `json:"id"`
	Status         Status    `json:"status"`
	LastHeartbeat  time.Time `json:"last_heartbeat"`
	StartedAt      time.Time `json:"started_at"`
	MessagesSent   int64     `json:"messages_sent"`
	MessagesFailed int64     `json:"messages_failed"`
	Alive          bool      `json:"alive"`
}

func ListAgents(ctx context.Context, rdb *redis.Client) ([]AgentSnapshot, error) {
	keys, err := rdb.Keys(ctx, keyPrefix+"*").Result()
	if err != nil {
		return nil, fmt.Errorf("list agents: %w", err)
	}

	agents := make([]AgentSnapshot, 0, len(keys))
	for _, key := range keys {
		data, err := rdb.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var info AgentInfo
		if err := json.Unmarshal(data, &info); err != nil {
			continue
		}

		alive := time.Since(info.LastHeartbeat) < heartbeatTTL
		agents = append(agents, AgentSnapshot{
			ID:             info.ID,
			Status:         info.Status,
			LastHeartbeat:  info.LastHeartbeat,
			StartedAt:      info.StartedAt,
			MessagesSent:   info.MessagesSent,
			MessagesFailed: info.MessagesFailed,
			Alive:          alive,
		})
	}

	return agents, nil
}

func GetAgentCount(ctx context.Context, rdb *redis.Client) (active, total int, err error) {
	agents, err := ListAgents(ctx, rdb)
	if err != nil {
		return 0, 0, err
	}

	total = len(agents)
	for _, a := range agents {
		if a.Alive {
			active++
		}
	}

	return active, total, nil
}

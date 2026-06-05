package network

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

type HeartbeatClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
	offlineQ   []map[string]interface{}
	mu         sync.Mutex
	maxQueue   int
	agentID    string
}

func NewHeartbeatClient(baseURL, apiKey string, timeout int, maxQueue int) *HeartbeatClient {
	return &HeartbeatClient{
		baseURL: baseURL,
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: time.Duration(timeout) * time.Second,
		},
		maxQueue: maxQueue,
		agentID:  uuid.New().String(),
	}
}

func (c *HeartbeatClient) SendHeartbeat(data map[string]interface{}) error {
	data["agent_id"] = c.agentID
	data["timestamp"] = time.Now().UTC().Format(time.RFC3339)

	body, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal error: %w", err)
	}

	req, err := http.NewRequest("POST", c.baseURL+"/api/v1/devices/heartbeat", bytes.NewReader(body))
	if err != nil {
		return c.queue(data)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return c.queue(data)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	respBody, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("heartbeat failed: %d %s", resp.StatusCode, string(respBody))
}

func (c *HeartbeatClient) queue(data map[string]interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.offlineQ) >= c.maxQueue {
		c.offlineQ = c.offlineQ[1:]
	}

	c.offlineQ = append(c.offlineQ, data)
	return fmt.Errorf("offline queued")
}

func (c *HeartbeatClient) FlushQueue() int {
	c.mu.Lock()
	queue := make([]map[string]interface{}, len(c.offlineQ))
	copy(queue, c.offlineQ)
	c.offlineQ = c.offlineQ[:0]
	c.mu.Unlock()

	sent := 0
	for _, data := range queue {
		body, _ := json.Marshal(data)
		req, _ := http.NewRequest("POST", c.baseURL+"/api/v1/devices/heartbeat", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if c.apiKey != "" {
			req.Header.Set("X-API-Key", c.apiKey)
		}

		resp, err := c.httpClient.Do(req)
		if err == nil && resp.StatusCode >= 200 && resp.StatusCode < 300 {
			sent++
		}
		if resp != nil {
			resp.Body.Close()
		}
	}

	return sent
}

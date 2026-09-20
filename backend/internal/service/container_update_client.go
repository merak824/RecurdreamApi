package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"time"
)

var ErrContainerUpdateAgentUnavailable = errors.New("update agent unavailable or not configured")

type UpdateExecution struct {
	OperationID string `json:"operation_id"`
	Version     string `json:"version"`
	Status      string `json:"status"`
	Message     string `json:"message,omitempty"`
	NeedRestart bool   `json:"need_restart"`
}

type ContainerUpdateClient interface {
	Start(ctx context.Context, version string) (*UpdateExecution, error)
	Status(ctx context.Context, operationID string) (*UpdateExecution, error)
}

type containerUpdateClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewContainerUpdateClient(socketPath string) (ContainerUpdateClient, error) {
	socketPath = strings.TrimSpace(socketPath)
	if socketPath == "" {
		return nil, ErrContainerUpdateAgentUnavailable
	}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{Timeout: 5 * time.Second}).DialContext(ctx, "unix", socketPath)
		},
	}
	return &containerUpdateClient{
		baseURL: "http://update-agent",
		httpClient: &http.Client{
			Transport: transport,
			Timeout:   15 * time.Second,
		},
	}, nil
}

func NewContainerUpdateClientFromEnv() ContainerUpdateClient {
	socketPath := strings.TrimSpace(os.Getenv("UPDATE_AGENT_SOCKET"))
	if socketPath == "" {
		socketPath = "/run/recurdream/update-agent.sock"
	}
	client, err := NewContainerUpdateClient(socketPath)
	if err != nil {
		return nil
	}
	return client
}

func newContainerUpdateClientFromHTTP(socketPath string, httpClient *http.Client) (*containerUpdateClient, error) {
	if strings.TrimSpace(socketPath) == "" || httpClient == nil {
		return nil, ErrContainerUpdateAgentUnavailable
	}
	return &containerUpdateClient{baseURL: "http://update-agent", httpClient: httpClient}, nil
}

func (c *containerUpdateClient) Start(ctx context.Context, version string) (*UpdateExecution, error) {
	body, err := json.Marshal(map[string]string{"version": version})
	if err != nil {
		return nil, err
	}
	return c.do(ctx, http.MethodPost, "/v1/update", body)
}

func (c *containerUpdateClient) Status(ctx context.Context, operationID string) (*UpdateExecution, error) {
	if strings.TrimSpace(operationID) == "" || strings.Contains(operationID, "/") {
		return nil, fmt.Errorf("invalid update operation id")
	}
	return c.do(ctx, http.MethodGet, "/v1/update/"+operationID, nil)
}

func (c *containerUpdateClient) do(ctx context.Context, method, path string, body []byte) (*UpdateExecution, error) {
	request, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	response, err := c.httpClient.Do(request)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrContainerUpdateAgentUnavailable, err)
	}
	defer func() { _ = response.Body.Close() }()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		var errorResponse struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(responseBody, &errorResponse) == nil && strings.TrimSpace(errorResponse.Message) != "" {
			return nil, errors.New(errorResponse.Message)
		}
		return nil, fmt.Errorf("update agent returned HTTP %d", response.StatusCode)
	}
	var execution UpdateExecution
	if err := json.Unmarshal(responseBody, &execution); err != nil {
		return nil, fmt.Errorf("decode update agent response: %w", err)
	}
	return &execution, nil
}

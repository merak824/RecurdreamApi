package service

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type containerUpdateRoundTripper struct {
	statusCode int
	response   string
	method     string
	path       string
	body       string
}

func (t *containerUpdateRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	t.method = req.Method
	t.path = req.URL.Path
	t.body = string(body)
	return &http.Response{
		StatusCode: t.statusCode,
		Status:     http.StatusText(t.statusCode),
		Body:       io.NopCloser(strings.NewReader(t.response)),
		Header:     make(http.Header),
		Request:    req,
	}, nil
}

func newTestContainerUpdateClient(roundTripper http.RoundTripper) *containerUpdateClient {
	return &containerUpdateClient{
		baseURL:    "http://update-agent",
		httpClient: &http.Client{Transport: roundTripper},
	}
}

func TestContainerUpdateClientStartSendsVersion(t *testing.T) {
	transport := &containerUpdateRoundTripper{
		statusCode: http.StatusAccepted,
		response:   `{"operation_id":"update-1","version":"0.2.5","status":"queued"}`,
	}
	client := newTestContainerUpdateClient(transport)

	execution, err := client.Start(context.Background(), "0.2.5")

	require.NoError(t, err)
	require.Equal(t, "update-1", execution.OperationID)
	require.Equal(t, "0.2.5", execution.Version)
	require.Equal(t, "queued", execution.Status)
	require.Equal(t, http.MethodPost, transport.method)
	require.Equal(t, "/v1/update", transport.path)
	require.JSONEq(t, `{"version":"0.2.5"}`, transport.body)
}

func TestContainerUpdateClientStatusReadsOperation(t *testing.T) {
	transport := &containerUpdateRoundTripper{
		statusCode: http.StatusOK,
		response:   `{"operation_id":"update-1","version":"0.2.5","status":"succeeded","message":"deployment completed"}`,
	}
	client := newTestContainerUpdateClient(transport)

	execution, err := client.Status(context.Background(), "update-1")

	require.NoError(t, err)
	require.Equal(t, "succeeded", execution.Status)
	require.Equal(t, http.MethodGet, transport.method)
	require.Equal(t, "/v1/update/update-1", transport.path)
}

func TestContainerUpdateClientReturnsAgentError(t *testing.T) {
	transport := &containerUpdateRoundTripper{
		statusCode: http.StatusServiceUnavailable,
		response:   `{"message":"update agent unavailable"}`,
	}
	client := newTestContainerUpdateClient(transport)

	_, err := client.Start(context.Background(), "0.2.5")

	require.Error(t, err)
	require.Contains(t, err.Error(), "update agent unavailable")
}

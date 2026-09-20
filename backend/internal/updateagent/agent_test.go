package updateagent

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type fakeRunner struct {
	mu       sync.Mutex
	versions []string
	started  chan struct{}
	release  chan struct{}
	err      error
}

func (r *fakeRunner) Run(ctx context.Context, _ Config, version string, _ io.Writer) error {
	r.mu.Lock()
	r.versions = append(r.versions, version)
	r.mu.Unlock()
	if r.started != nil {
		select {
		case r.started <- struct{}{}:
		default:
		}
	}
	if r.release != nil {
		select {
		case <-r.release:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return r.err
}

func newTestAgent(t *testing.T, runner Runner) (*Agent, string) {
	t.Helper()
	root := t.TempDir()
	versionFile := filepath.Join(root, "current-version")
	require.NoError(t, os.WriteFile(versionFile, []byte("0.1.177\n"), 0o600))
	agent, err := New(Config{
		DeployPath:         root,
		HotDeployScript:    filepath.Join(root, "hot-deploy.sh"),
		ImageRepository:    "ghcr.io/merak824/recurdreamapi",
		CurrentVersionFile: versionFile,
		LogDir:             filepath.Join(root, "logs"),
		DrainSeconds:       300,
		Timeout:            time.Minute,
	}, runner)
	require.NoError(t, err)
	return agent, versionFile
}

func postUpdate(t *testing.T, handler http.Handler, version string) (int, Operation) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/v1/update", httptest.NewRequest(http.MethodPost, "/", nil).Body)
	req.Body = io.NopCloser(bytesReader(`{"version":"` + version + `"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	var operation Operation
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &operation))
	return rec.Code, operation
}

func bytesReader(value string) io.Reader {
	return &stringReader{value: value}
}

type stringReader struct {
	value string
	read  bool
}

func (r *stringReader) Read(p []byte) (int, error) {
	if r.read {
		return 0, io.EOF
	}
	r.read = true
	return copy(p, r.value), nil
}

func waitForOperation(t *testing.T, handler http.Handler, operationID string) Operation {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		req := httptest.NewRequest(http.MethodGet, "/v1/update/"+operationID, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		var operation Operation
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &operation))
		if operation.Status == StatusSucceeded || operation.Status == StatusFailed {
			return operation
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("operation %s did not finish", operationID)
	return Operation{}
}

func TestAgentRejectsInvalidVersion(t *testing.T) {
	agent, _ := newTestAgent(t, &fakeRunner{})

	status, operation := postUpdate(t, agent.Handler(), "../../etc/passwd")

	require.Equal(t, http.StatusBadRequest, status)
	require.Equal(t, "", operation.OperationID)
}

func TestAgentRequiresFixedDrainWindow(t *testing.T) {
	_, err := New(Config{
		ImageRepository: "ghcr.io/merak824/recurdreamapi",
		DrainSeconds:    299,
		Timeout:         time.Minute,
	}, &fakeRunner{})

	require.ErrorContains(t, err, "drain seconds must be 300")
}

func TestAgentRunsFixedVersionAndPersistsItAfterSuccess(t *testing.T) {
	runner := &fakeRunner{started: make(chan struct{}, 1), release: make(chan struct{})}
	agent, versionFile := newTestAgent(t, runner)

	status, queued := postUpdate(t, agent.Handler(), "0.2.5")
	require.Equal(t, http.StatusAccepted, status)
	require.Equal(t, "0.2.5", queued.Version)
	<-runner.started
	close(runner.release)

	finished := waitForOperation(t, agent.Handler(), queued.OperationID)
	require.Equal(t, StatusSucceeded, finished.Status)
	require.Equal(t, "0.2.5", strings.TrimSpace(string(mustReadFile(t, versionFile))))
	runner.mu.Lock()
	require.Equal(t, []string{"0.2.5"}, runner.versions)
	runner.mu.Unlock()
}

func TestAgentPreventsConcurrentUpdatesAndKeepsVersionOnFailure(t *testing.T) {
	runner := &fakeRunner{started: make(chan struct{}, 1), release: make(chan struct{})}
	agent, versionFile := newTestAgent(t, runner)

	status, first := postUpdate(t, agent.Handler(), "0.2.5")
	require.Equal(t, http.StatusAccepted, status)
	<-runner.started

	status, _ = postUpdate(t, agent.Handler(), "0.2.6")
	require.Equal(t, http.StatusConflict, status)

	close(runner.release)
	finished := waitForOperation(t, agent.Handler(), first.OperationID)
	require.Equal(t, StatusSucceeded, finished.Status)
	require.Equal(t, "0.2.5", strings.TrimSpace(string(mustReadFile(t, versionFile))))
}

func TestAgentDoesNotAdvanceVersionWhenRunnerFails(t *testing.T) {
	runner := &fakeRunner{err: os.ErrPermission}
	agent, versionFile := newTestAgent(t, runner)

	status, queued := postUpdate(t, agent.Handler(), "0.2.5")
	require.Equal(t, http.StatusAccepted, status)
	finished := waitForOperation(t, agent.Handler(), queued.OperationID)

	require.Equal(t, StatusFailed, finished.Status)
	require.Contains(t, finished.Message, "permission denied")
	require.Equal(t, "0.1.177", strings.TrimSpace(string(mustReadFile(t, versionFile))))
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	require.NoError(t, err)
	return data
}

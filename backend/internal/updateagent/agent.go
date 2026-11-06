package updateagent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusSucceeded = "succeeded"
	StatusFailed    = "failed"
)

var (
	ErrUpdateInProgress    = errors.New("another update is already running")
	ErrVersionNotNewer     = errors.New("requested version is not newer than the deployed version")
	semverPattern          = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)
	imageRepositoryPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._/-]*$`)
)

type Config struct {
	SocketPath         string
	DeployPath         string
	HotDeployScript    string
	ImageRepository    string
	CurrentVersionFile string
	LogDir             string
	DrainSeconds       int
	Timeout            time.Duration
}

type Runner interface {
	Run(ctx context.Context, cfg Config, version string, logWriter io.Writer) error
}

type CommandRunner struct{}

type Operation struct {
	OperationID string     `json:"operation_id"`
	Version     string     `json:"version"`
	Status      string     `json:"status"`
	Message     string     `json:"message,omitempty"`
	StartedAt   *time.Time `json:"started_at,omitempty"`
	FinishedAt  *time.Time `json:"finished_at,omitempty"`
	LogPath     string     `json:"log_path,omitempty"`
}

type Agent struct {
	cfg        Config
	runner     Runner
	mu         sync.Mutex
	activeID   string
	operations map[string]*Operation
}

func New(cfg Config, runner Runner) (*Agent, error) {
	if cfg.DrainSeconds == 0 {
		cfg.DrainSeconds = 300
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 30 * time.Minute
	}
	if strings.TrimSpace(cfg.ImageRepository) == "" || !imageRepositoryPattern.MatchString(cfg.ImageRepository) {
		return nil, fmt.Errorf("invalid image repository")
	}
	if cfg.DrainSeconds < 0 {
		return nil, fmt.Errorf("drain seconds cannot be negative")
	}
	if cfg.DrainSeconds != 300 {
		return nil, fmt.Errorf("drain seconds must be 300")
	}
	if cfg.Timeout <= 0 {
		return nil, fmt.Errorf("agent timeout must be positive")
	}
	if runner == nil {
		runner = CommandRunner{}
	}
	return &Agent{
		cfg:        cfg,
		runner:     runner,
		operations: make(map[string]*Operation),
	}, nil
}

func (CommandRunner) Run(ctx context.Context, cfg Config, version string, logWriter io.Writer) error {
	image := cfg.ImageRepository + ":" + version
	cmd := exec.CommandContext(ctx, "/bin/bash", cfg.HotDeployScript, "--image", image)
	cmd.Dir = cfg.DeployPath
	cmd.Env = append(os.Environ(),
		"DRAIN_SECONDS="+strconv.Itoa(cfg.DrainSeconds),
		"KEEP_OLD=false",
		"NO_PULL=false",
	)
	cmd.Stdout = logWriter
	cmd.Stderr = logWriter
	return cmd.Run()
}

func (a *Agent) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/update", a.handleStart)
	mux.HandleFunc("/v1/update/", a.handleStatus)
	return mux
}

func (a *Agent) handleStart(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var request struct {
		Version string `json:"version"`
	}
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, "invalid update request")
		return
	}
	if !semverPattern.MatchString(request.Version) {
		writeError(w, http.StatusBadRequest, "version must use major.minor.patch format")
		return
	}

	operation, reused, err := a.enqueue(request.Version)
	if err != nil {
		status := http.StatusBadRequest
		if errors.Is(err, ErrUpdateInProgress) || errors.Is(err, ErrVersionNotNewer) {
			status = http.StatusConflict
		}
		writeError(w, status, err.Error())
		return
	}
	if reused {
		writeJSON(w, http.StatusOK, operation)
		return
	}
	writeJSON(w, http.StatusAccepted, operation)
}

func (a *Agent) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	operationID := strings.TrimPrefix(r.URL.Path, "/v1/update/")
	if operationID == "" || strings.Contains(operationID, "/") {
		writeError(w, http.StatusBadRequest, "invalid operation id")
		return
	}
	a.mu.Lock()
	operation, ok := a.operations[operationID]
	if ok {
		operation = cloneOperation(operation)
	}
	a.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, "update operation not found")
		return
	}
	writeJSON(w, http.StatusOK, operation)
}

func (a *Agent) enqueue(version string) (*Operation, bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.activeID != "" {
		active := a.operations[a.activeID]
		if active != nil && (active.Status == StatusQueued || active.Status == StatusRunning) {
			if active.Version == version {
				return cloneOperation(active), true, nil
			}
			return nil, false, ErrUpdateInProgress
		}
	}

	currentVersion, err := a.readCurrentVersion()
	if err != nil {
		return nil, false, fmt.Errorf("read deployed version: %w", err)
	}
	if compareVersions(version, currentVersion) <= 0 {
		return nil, false, ErrVersionNotNewer
	}

	now := time.Now().UTC()
	operation := &Operation{
		OperationID: "update-" + strconv.FormatInt(now.UnixNano(), 36),
		Version:     version,
		Status:      StatusQueued,
	}
	a.operations[operation.OperationID] = operation
	a.activeID = operation.OperationID
	go a.execute(operation.OperationID)
	return cloneOperation(operation), false, nil
}

func (a *Agent) execute(operationID string) {
	a.setStatus(operationID, StatusRunning, "update started")

	a.mu.Lock()
	operation := cloneOperation(a.operations[operationID])
	a.mu.Unlock()

	var logWriter = io.Discard
	var logFile *os.File
	if a.cfg.LogDir != "" {
		if err := os.MkdirAll(a.cfg.LogDir, 0o750); err == nil {
			logPath := filepath.Join(a.cfg.LogDir, operationID+".log")
			if file, createErr := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o640); createErr == nil {
				logFile = file
				logWriter = file
				a.mu.Lock()
				if current := a.operations[operationID]; current != nil {
					current.LogPath = logPath
				}
				a.mu.Unlock()
			}
		}
	}
	if logFile != nil {
		defer func() { _ = logFile.Close() }()
	}

	ctx, cancel := context.WithTimeout(context.Background(), a.cfg.Timeout)
	err := a.runner.Run(ctx, a.cfg, operation.Version, logWriter)
	cancel()
	if err != nil {
		a.setStatus(operationID, StatusFailed, fmt.Sprintf("deployment failed: %v", err))
		a.finish(operationID)
		return
	}
	if err := a.persistCurrentVersion(operation.Version); err != nil {
		a.setStatus(operationID, StatusFailed, fmt.Sprintf("deployment succeeded but version state could not be saved: %v", err))
		a.finish(operationID)
		return
	}
	a.setStatus(operationID, StatusSucceeded, "deployment completed")
	a.finish(operationID)
}

func (a *Agent) finish(operationID string) {
	now := time.Now().UTC()
	a.mu.Lock()
	defer a.mu.Unlock()
	if operation := a.operations[operationID]; operation != nil {
		operation.FinishedAt = &now
	}
	if a.activeID == operationID {
		a.activeID = ""
	}
}

func (a *Agent) setStatus(operationID, status, message string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if operation := a.operations[operationID]; operation != nil {
		operation.Status = status
		operation.Message = message
		if status == StatusRunning {
			now := time.Now().UTC()
			operation.StartedAt = &now
		}
	}
}

func (a *Agent) readCurrentVersion() (string, error) {
	if strings.TrimSpace(a.cfg.CurrentVersionFile) == "" {
		return "0.0.0", nil
	}
	data, err := os.ReadFile(a.cfg.CurrentVersionFile)
	if err != nil {
		return "", err
	}
	version := strings.TrimSpace(string(data))
	if !semverPattern.MatchString(version) {
		return "", fmt.Errorf("invalid deployed version %q", version)
	}
	return version, nil
}

func (a *Agent) persistCurrentVersion(version string) error {
	if strings.TrimSpace(a.cfg.CurrentVersionFile) == "" {
		return nil
	}
	dir := filepath.Dir(a.cfg.CurrentVersionFile)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".update-agent-version-*")
	if err != nil {
		return err
	}
	tempName := temp.Name()
	defer func() { _ = os.Remove(tempName) }()
	if err := temp.Chmod(0o600); err != nil {
		_ = temp.Close()
		return err
	}
	if _, err := temp.WriteString(version + "\n"); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(tempName, a.cfg.CurrentVersionFile)
}

func cloneOperation(operation *Operation) *Operation {
	if operation == nil {
		return nil
	}
	copy := *operation
	return &copy
}

func compareVersions(left, right string) int {
	leftParts := strings.Split(left, ".")
	rightParts := strings.Split(right, ".")
	for i := 0; i < 3; i++ {
		leftNumber, _ := strconv.Atoi(leftParts[i])
		rightNumber, _ := strconv.Atoi(rightParts[i])
		if leftNumber < rightNumber {
			return -1
		}
		if leftNumber > rightNumber {
			return 1
		}
	}
	return 0
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"message": message})
}

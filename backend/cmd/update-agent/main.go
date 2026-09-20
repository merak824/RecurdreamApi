package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/updateagent"
)

func main() {
	cfg, err := loadConfig()
	if err != nil {
		slog.Error("invalid update agent configuration", "error", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(filepath.Dir(cfg.SocketPath), 0o750); err != nil {
		slog.Error("create socket directory", "error", err)
		os.Exit(1)
	}
	if err := removeStaleSocket(cfg.SocketPath); err != nil {
		slog.Error("remove stale socket", "error", err)
		os.Exit(1)
	}

	agent, err := updateagent.New(cfg, nil)
	if err != nil {
		slog.Error("create update agent", "error", err)
		os.Exit(1)
	}
	listener, err := net.Listen("unix", cfg.SocketPath)
	if err != nil {
		slog.Error("listen on update agent socket", "error", err)
		os.Exit(1)
	}
	defer func() { _ = os.Remove(cfg.SocketPath) }()
	if err := os.Chmod(cfg.SocketPath, 0o660); err != nil {
		slog.Error("set update agent socket permissions", "error", err)
		os.Exit(1)
	}

	server := &http.Server{Handler: agent.Handler(), ReadHeaderTimeout: 5 * time.Second}
	serveErr := make(chan error, 1)
	go func() { serveErr <- server.Serve(listener) }()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			slog.Error("shutdown update agent", "error", err)
		}
	case err := <-serveErr:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("update agent stopped", "error", err)
			os.Exit(1)
		}
	}
}

func loadConfig() (updateagent.Config, error) {
	drainSeconds, err := envInt("UPDATE_AGENT_DRAIN_SECONDS", 300)
	if err != nil {
		return updateagent.Config{}, err
	}
	timeoutMinutes, err := envInt("UPDATE_AGENT_TIMEOUT_MINUTES", 30)
	if err != nil {
		return updateagent.Config{}, err
	}
	return updateagent.Config{
		SocketPath:         envString("UPDATE_AGENT_SOCKET", "/run/recurdream/update-agent.sock"),
		DeployPath:         envString("UPDATE_AGENT_DEPLOY_PATH", "/data/sub2api/deploy"),
		HotDeployScript:    envString("UPDATE_AGENT_HOT_DEPLOY_SCRIPT", "/data/sub2api/deploy/hot-deploy.sh"),
		ImageRepository:    envString("UPDATE_AGENT_IMAGE", "ghcr.io/merak824/recurdreamapi"),
		CurrentVersionFile: envString("UPDATE_AGENT_CURRENT_VERSION_FILE", "/data/sub2api/.update-agent/current-version"),
		LogDir:             envString("UPDATE_AGENT_LOG_DIR", "/var/log/recurdream-update-agent"),
		DrainSeconds:       drainSeconds,
		Timeout:            time.Duration(timeoutMinutes) * time.Minute,
	}, nil
}

func removeStaleSocket(path string) error {
	info, err := os.Lstat(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("refusing to remove non-socket path %s", path)
	}
	return os.Remove(path)
}

func envString(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func envInt(name string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", name)
	}
	return parsed, nil
}

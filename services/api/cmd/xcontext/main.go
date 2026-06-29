package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/agumbe-ai/xcontext/services/api/internal/api"
	"github.com/agumbe-ai/xcontext/services/api/internal/config"
	"github.com/agumbe-ai/xcontext/services/api/internal/models"
	"github.com/agumbe-ai/xcontext/services/api/internal/service"
	"github.com/agumbe-ai/xcontext/services/api/internal/store"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		panic(err)
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	st := store.NewMemory()
	svc := service.New(st, service.Config{CostPer1K: cfg.CostPer1K, StoreRawMode: cfg.StoreRawMode, MaxInputBytes: cfg.MaxInputBytes, MaxSummaryTokens: cfg.MaxSummaryTokens, ConsoleURL: cfg.ConsoleURL})
	auth := api.DevScopeResolver{Enabled: cfg.DevAuth, Scope: models.Scope{TenantID: cfg.DevTenantID, WorkspaceID: cfg.DevWorkspaceID, UserID: cfg.DevUserID, TrustedInterceptor: cfg.DevTrustedInterceptor}}
	server := http.Server{Addr: ":" + cfg.Port, Handler: api.New(svc, auth, cfg.BasePath, log), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 30 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 1 << 20}
	log.Info("xcontext listening", "address", server.Addr, "devAuth", cfg.DevAuth)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		panic(err)
	}
}

package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port, BasePath, StoreRawMode, ConsoleURL string
	CostPer1K                                float64
	MaxInputBytes, MaxSummaryTokens          int
	DevAuth                                  bool
	DevTrustedInterceptor                    bool
	DevTenantID, DevWorkspaceID, DevUserID   string
}

func Load() (Config, error) {
	c := Config{Port: get("XCONTEXT_PORT", "8080"), BasePath: get("XCONTEXT_API_BASE_PATH", "/xcontext/v1"), StoreRawMode: get("XCONTEXT_STORE_RAW_MODE", "redacted"), ConsoleURL: get("XCONTEXT_CONSOLE_URL", "https://console.agumbe.ai/xcontext"), CostPer1K: getFloat("XCONTEXT_ESTIMATED_COST_PER_1K_TOKENS", .01), MaxInputBytes: getInt("XCONTEXT_MAX_INPUT_BYTES", 10<<20), MaxSummaryTokens: getInt("XCONTEXT_MAX_SUMMARY_TOKENS", 1200), DevAuth: strings.EqualFold(os.Getenv("XCONTEXT_DEV_AUTH_ENABLED"), "true"), DevTrustedInterceptor: strings.EqualFold(os.Getenv("XCONTEXT_DEV_TRUSTED_INTERCEPTOR"), "true"), DevTenantID: os.Getenv("XCONTEXT_DEV_TENANT_ID"), DevWorkspaceID: os.Getenv("XCONTEXT_DEV_WORKSPACE_ID"), DevUserID: os.Getenv("XCONTEXT_DEV_USER_ID")}
	if c.StoreRawMode != "redacted" && c.StoreRawMode != "original" && c.StoreRawMode != "none" {
		return c, fmt.Errorf("invalid XCONTEXT_STORE_RAW_MODE")
	}
	if c.DevAuth && (c.DevTenantID == "" || c.DevWorkspaceID == "") {
		return c, fmt.Errorf("dev auth requires tenant and workspace IDs")
	}
	return c, nil
}
func get(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}
func getInt(k string, d int) int {
	if v, e := strconv.Atoi(os.Getenv(k)); e == nil && v > 0 {
		return v
	}
	return d
}
func getFloat(k string, d float64) float64 {
	if v, e := strconv.ParseFloat(os.Getenv(k), 64); e == nil && v >= 0 {
		return v
	}
	return d
}

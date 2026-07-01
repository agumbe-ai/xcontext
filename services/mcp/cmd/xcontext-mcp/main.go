package main

import (
	"context"
	"fmt"
	"os"

	"github.com/agumbe-ai/xcontext/pkg/client"
	"github.com/agumbe-ai/xcontext/services/mcp/internal/server"
)

func main() {
	url := os.Getenv("AGUMBE_XCONTEXT_API_URL")
	if url == "" {
		url = "https://api.agumbe.ai/xcontext/v1"
	}
	key := os.Getenv("AGUMBE_XCONTEXT_API_KEY")
	if key == "" {
		fmt.Fprintln(os.Stderr, "AGUMBE_XCONTEXT_API_KEY is required")
		os.Exit(2)
	}
	if e := server.New(client.New(url, key), os.Stdin, os.Stdout).Run(context.Background()); e != nil {
		fmt.Fprintln(os.Stderr, e)
		os.Exit(1)
	}
}

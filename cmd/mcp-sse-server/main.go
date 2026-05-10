package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	spMcp "github.com/superplanehq/superplane/pkg/mcp"
	"github.com/superplanehq/superplane/pkg/mcp/tools"
)

type loggingHandler struct {
	handler http.Handler
}

func (h *loggingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Printf("[HTTP] %s %s content-length=%d from=%s", r.Method, r.URL.String(), r.ContentLength, r.RemoteAddr)
	h.handler.ServeHTTP(w, r)
}

func main() {
	config, err := spMcp.LoadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	apiClient := spMcp.NewAPIClient(config)
	bearerToken := os.Getenv("MCP_BEARER_TOKEN")
	port := os.Getenv("MCP_PORT")
	if port == "" {
		port = "8484"
	}

	handler := mcp.NewSSEHandler(func(r *http.Request) *mcp.Server {
		log.Printf("[MCP] New SSE session from %s", r.RemoteAddr)

		// Auth check
		if bearerToken != "" {
			auth := r.Header.Get("Authorization")
			if auth != "Bearer "+bearerToken {
				log.Printf("[MCP] Auth failed from %s", r.RemoteAddr)
				return nil
			}
		}

		s := mcp.NewServer(&mcp.Implementation{
			Name:    "superplane",
			Version: "0.1.0",
		}, nil)

		ctx := context.Background()
		tools.RegisterCanvasTools(ctx, s, apiClient)
		tools.RegisterCanvasMutationTools(ctx, s, apiClient)
		tools.RegisterEventTools(ctx, s, apiClient)
		tools.RegisterExecutionTools(ctx, s, apiClient)
		tools.RegisterIntegrationTools(ctx, s, apiClient)
		tools.RegisterSecretTools(ctx, s, apiClient)
		tools.RegisterOperationalTools(ctx, s, apiClient)
		tools.RegisterDiscoveryTools(ctx, s, apiClient)

		return s
	}, &mcp.SSEOptions{
		DisableLocalhostProtection: true,
	})

	fmt.Printf("SuperPlane MCP SSE server starting on :%s\n", port)
	fmt.Printf("Endpoint: http://0.0.0.0:%s/sse\n", port)
	fmt.Printf("Auth: %v\n", bearerToken != "")

	mux := http.NewServeMux()
	mux.Handle("/", &loggingHandler{handler: handler})

	log.Fatal(http.ListenAndServe(":"+port, mux))
}

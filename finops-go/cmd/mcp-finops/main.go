// Command mcp-finops runs the MCP tool server for FinOps workflow operations.
// Uses stdio transport for integration with AI assistants.
package main

import (
	"context"
	"log"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"go.temporal.io/sdk/client"

	"github.com/finops-claw-gang/finops-go/internal/mcpserver"
	"github.com/finops-claw-gang/finops-go/internal/temporal/querier"
)

func main() {
	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalf("unable to create Temporal client: %v", err)
	}
	defer c.Close()

	q := querier.New(c)

	server := mcp.NewServer(&mcp.Implementation{
		Name:    "finops-claw-gang",
		Version: "v1.0.0",
	}, nil)
	mcpserver.RegisterTools(server, q)

	if err := server.Run(context.Background(), &mcp.StdioTransport{}); err != nil {
		log.Fatalf("mcp server error: %v", err)
	}
}

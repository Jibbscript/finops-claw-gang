// Command api runs the HTTP API server for the FinOps Generative UI.
package main

import (
	"log"
	"net/http"

	"go.temporal.io/sdk/client"

	"github.com/finops-claw-gang/finops-go/internal/api"
	"github.com/finops-claw-gang/finops-go/internal/config"
	"github.com/finops-claw-gang/finops-go/internal/temporal/querier"
)

func main() {
	cfg, err := config.LoadFromEnv()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	c, err := client.Dial(client.Options{})
	if err != nil {
		log.Fatalf("unable to create Temporal client: %v", err)
	}
	defer c.Close()

	q := querier.New(c)
	srv := api.New(q, cfg.CORSOrigins)

	addr := ":" + cfg.APIPort
	log.Printf("starting API server on %s", addr)
	if err := http.ListenAndServe(addr, srv); err != nil {
		log.Fatalf("server error: %v", err)
	}
}

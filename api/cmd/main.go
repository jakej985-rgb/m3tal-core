package main

import (
	"log"
	"os"

	"github.com/jakej985-rgb/m3tal-core/api"
	syspaths "github.com/jakej985-rgb/m3tal-core/pkg/system"
)

func main() {
	syspaths.SetProcessName("m3tal-api")
	unlock, acquired := syspaths.AcquireProcessLock("m3tal-api")
	if !acquired {
		log.Println("⚠️ m3tal-api is already running. Exiting duplicate instance.")
		return
	}
	defer unlock()

	apiToken := os.Getenv("API_TOKEN")
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "5050"
	}

	if err := api.RunServer(port, apiToken); err != nil {
		log.Fatalf("❌ API server failed: %v", err)
	}
}

package main

import (
	"context"
	"fmt"
	"organizer/internal/audit"
	"organizer/internal/copier"
	"os"
	"sync"

	"organizer/internal/ai"
	"organizer/internal/analyzer"
	"organizer/internal/configuration"
	"organizer/internal/scanner"
)

func main() {

	ctx := context.Background()

	waitGroup := &sync.WaitGroup{}

	auditService, err := audit.New()

	if err != nil {
		fmt.Printf("Unable to initialize the audit service: %v\n", err)
		os.Exit(1)
	}

	//	Initializes the configuration service
	configurationService, err := configuration.New()

	if err != nil {
		fmt.Printf("Unable to initialize the configuration service: %v\n", err)
		os.Exit(1)
	}

	//	Initializes the AI proxy
	aiProxy, err := ai.New(configurationService, ctx)

	if err != nil {
		fmt.Printf("Unable to start the AI proxy: %v\n", err)
		os.Exit(1)
	}

	scannerService := scanner.New(configurationService, aiProxy, auditService, ctx, waitGroup)
	analyzerService := analyzer.New(aiProxy, scannerService, auditService, ctx, waitGroup)
	copierService := copier.New(configurationService, analyzerService, auditService, ctx, waitGroup)

	//	Runs the application
	scannerService.Scan()
	analyzerService.Run()
	copierService.Run()

	waitGroup.Wait()
}

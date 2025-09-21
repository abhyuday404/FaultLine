package main

// TROUBLESHOOTING: If you see an "undefined: api.RegisterHandlers" error,
// please check your go.mod file. The first line MUST be exactly:
// module faultline
//
// If it's different, please change it. After saving the change, run this
// command in your terminal:
// go mod tidy
//
// This will resolve the error by aligning your project's module name with
// the import paths used in the code.

import (
	"context"
	"faultline/api"
	"faultline/proxy"
	"faultline/state"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
)

func main() {
	var proxyPort int
	var apiPort int

	var rootCmd = &cobra.Command{
		Use:   "faultline",
		Short: "A tool for injecting failure scenarios into your dev environment.",
	}

	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Starts the FaultLine proxy and control API servers",
		Run: func(cmd *cobra.Command, args []string) {
			runServers(apiPort, proxyPort)
		},
	}

	startCmd.Flags().IntVarP(&proxyPort, "proxy-port", "p", 8080, "Port for the failure injection proxy")
	startCmd.Flags().IntVarP(&apiPort, "api-port", "a", 8081, "Port for the control panel API")

	rootCmd.AddCommand(startCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

// runServers sets up and starts the API and proxy servers.
func runServers(apiPort, proxyPort int) {
	ruleState := state.NewRuleState(nil)

	// --- Setup Control API Server ---
	apiRouter := mux.NewRouter()
	api.RegisterHandlers(apiRouter, ruleState)

	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173", "http://localhost:5174"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	})
	apiHandler := c.Handler(apiRouter)

	apiServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", apiPort),
		Handler: apiHandler,
	}

	// --- Setup Proxy Server ---
	p := proxy.NewProxy(ruleState)
	proxyServer := &http.Server{
		Addr:    fmt.Sprintf(":%d", proxyPort),
		Handler: http.HandlerFunc(p.HandleRequest),
	}

	// --- Graceful Shutdown Setup ---
	stopChan := make(chan os.Signal, 1)
	signal.Notify(stopChan, syscall.SIGINT, syscall.SIGTERM)

	// --- Start Servers ---
	go func() {
		log.Printf("✅ Control API listening on http://localhost:%d", apiPort)
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("API server failed: %v", err)
		}
	}()

	go func() {
		log.Printf("✅ FaultLine Proxy listening on http://localhost:%d", proxyPort)
		if err := proxyServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Proxy server failed: %v", err)
		}
	}()

	// Block until a signal is received
	<-stopChan
	log.Println("Shutting down servers...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := apiServer.Shutdown(ctx); err != nil {
		log.Printf("API server shutdown error: %v", err)
	}
	if err := proxyServer.Shutdown(ctx); err != nil {
		log.Printf("Proxy server shutdown error: %v", err)
	}

	log.Println("Servers gracefully stopped.")
}

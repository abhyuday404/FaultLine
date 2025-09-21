package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"

	"faultline/api"
	"faultline/config"
	"faultline/proxy"
	"faultline/tcp"
	"faultline/state"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
)

func main() {
	var configFile string
	var port int
	var apiPort int

	var rootCmd = &cobra.Command{
		Use:   "faultline",
		Short: "FaultLine: all-in-one failure testing for APIs and Databases",
		Long: `FaultLine helps you build resilient apps by simulating real-world failures across:
 - API (HTTP) via a reverse proxy with latency/errors/flaky responses
 - DB (TCP) via a transparent proxy for network-level faults (latency, drops, throttling, refused)

Configure scenarios in a YAML file and run targeted commands to test each surface.`,
	}

	var startCmd = &cobra.Command{
		Use:   "start-api",
		Short: "Start API (HTTP) fault-injection proxy",
		Run: func(cmd *cobra.Command, args []string) {
			// 1. Load initial configuration from YAML
			initialConfig, err := config.LoadConfig(configFile)
			if err != nil {
				// We don't fail if the file doesn't exist, just start with no rules.
				log.Printf("Warning: No config file found at %s. Starting with zero rules.", configFile)
				initialConfig = &config.Config{}
			}

			// 2. Initialize the shared state for rules
			ruleState := state.NewRuleState(initialConfig.Rules)
			log.Printf("Loaded %d initial failure rules from %s", len(initialConfig.Rules), configFile)

			var wg sync.WaitGroup
			wg.Add(2)

			// 3. Start the Proxy Server in a separate goroutine
			go func() {
				defer wg.Done()
				p := proxy.NewProxy(ruleState)
				proxyRouter := mux.NewRouter()
				// The proxy handles all paths that are not for the API
				proxyRouter.PathPrefix("/").Handler(http.HandlerFunc(p.HandleRequest))

				log.Printf("-> FaultLine Proxy Server listening on port %d...", port)
				if err := http.ListenAndServe(fmt.Sprintf(":%d", port), proxyRouter); err != nil {
					log.Fatalf("Failed to start proxy server: %v", err)
				}
			}()

			// 4. Start the Control API Server in a separate goroutine
			go func() {
				defer wg.Done()
				apiHandler := api.NewApiHandler(ruleState)
				apiRouter := mux.NewRouter()

				// API routes
				apiRouter.HandleFunc("/api/rules", apiHandler.GetRules).Methods("GET")
				apiRouter.HandleFunc("/api/rules", apiHandler.AddRule).Methods("POST")
				apiRouter.HandleFunc("/api/rules/{id}", apiHandler.UpdateRule).Methods("PUT")
				apiRouter.HandleFunc("/api/rules/{id}", apiHandler.DeleteRule).Methods("DELETE")

				// CORS configuration
				c := cors.New(cors.Options{
					AllowedOrigins: []string{"*"}, // Allow all for development
					AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
					AllowedHeaders: []string{"Content-Type"},
				})
				handler := c.Handler(apiRouter)

				log.Printf("-> FaultLine Control API listening on port %d...", apiPort)
				if err := http.ListenAndServe(fmt.Sprintf(":%d", apiPort), handler); err != nil {
					log.Fatalf("Failed to start API server: %v", err)
				}
			}()

			// Wait for both servers to finish
			wg.Wait()
		},
	}

	startCmd.Flags().StringVarP(&configFile, "config", "c", "faultline.yaml", "Path to the initial configuration file")
	startCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port for the failure proxy server")
	startCmd.Flags().IntVarP(&apiPort, "api-port", "a", 8081, "Port for the control panel API")

	rootCmd.AddCommand(startCmd)

	// start-db command: start DB (TCP) fault injectors defined in tcpRules
	var startTCPCmd = &cobra.Command{
		Use:   "start-db",
		Short: "Start DB (TCP) fault-injection proxies from tcpRules",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.LoadConfig(configFile)
			if err != nil {
				log.Fatalf("Error loading config file: %v", err)
			}
			if len(cfg.TCPRules) == 0 {
				log.Println("[DB] No tcpRules found in config. Nothing to start.")
				return
			}

			stop := make(chan struct{})
			done := make(chan struct{})
			go func() {
				<-cmd.Context().Done()
				close(stop)
				close(done)
			}()

			for _, r := range cfg.TCPRules {
				rp := tcp.NewProxy(r)
				go func(rule config.TCPRule) {
					if err := rp.Start(stop); err != nil {
						log.Printf("[DB] Proxy %s -> %s exited: %v", rule.Listen, rule.Upstream, err)
					}
				}(r)
			}

			log.Printf("[DB] Started %d DB network proxies (latency/drops/throttle/refuse). Press Ctrl+C to stop.", len(cfg.TCPRules))
			<-done
		},
	}
	startTCPCmd.Flags().StringVarP(&configFile, "config", "c", "faultline.yaml", "Path to the configuration file")
	rootCmd.AddCommand(startTCPCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

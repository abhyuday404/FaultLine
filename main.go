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
	"faultline/state"
	"faultline/tcp"

	"github.com/gorilla/mux"
	"github.com/rs/cors"
	"github.com/spf13/cobra"
	yaml "gopkg.in/yaml.v2"
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

	// configure: interactive CLI to create/update faultline.yaml
	var configureCmd = &cobra.Command{
		Use:   "configure",
		Short: "Interactively create or update faultline.yaml",
		Run: func(cmd *cobra.Command, args []string) {
			// Simple interactive prompts using stdin
			var cfg config.Config
			if existing, err := config.LoadConfig(configFile); err == nil {
				cfg = *existing
				log.Printf("Loaded existing config with %d API rules and %d DB rules", len(cfg.Rules), len(cfg.TCPRules))
			}

			// Prompt helper
			ask := func(q string) string {
				fmt.Print(q)
				var s string
				fmt.Scanln(&s)
				return s
			}

			// API rules
			addAPI := ask("Add an API rule? (y/n): ")
			for addAPI == "y" || addAPI == "Y" {
				target := ask("  Target full URL (e.g., http://localhost:3000/users): ")
				ftype := ask("  Failure type [latency|error|flaky]: ")
				var f config.Failure
				f.Type = ftype
				switch ftype {
				case "latency":
					fmt.Print("  latency_ms (int): ")
					fmt.Scanln(&f.LatencyMs)
				case "error":
					fmt.Print("  error_code (int): ")
					fmt.Scanln(&f.ErrorCode)
				case "flaky":
					fmt.Print("  probability (0..1): ")
					fmt.Scanln(&f.Probability)
				}
				cfg.Rules = append(cfg.Rules, config.Rule{Target: target, Failure: f})
				addAPI = ask("Add another API rule? (y/n): ")
			}

			// DB rules
			addDB := ask("Add a DB/TCP rule? (y/n): ")
			for addDB == "y" || addDB == "Y" {
				listen := ask("  listen (e.g., 127.0.0.1:55432): ")
				upstream := ask("  upstream (e.g., localhost:5432): ")
				var faults config.TCPFaults
				fmt.Print("  latency_ms (int, 0 for none): ")
				fmt.Scanln(&faults.LatencyMs)
				fmt.Print("  drop_probability (0..1): ")
				fmt.Scanln(&faults.DropProbability)
				fmt.Print("  reset_probability (0..1): ")
				fmt.Scanln(&faults.ResetProbability)
				fmt.Print("  bandwidth_kbps (int, 0 for none): ")
				fmt.Scanln(&faults.BandwidthKbps)
				refuse := ask("  refuse_connections? (y/n): ")
				if refuse == "y" || refuse == "Y" {
					faults.RefuseConnections = true
				}
				cfg.TCPRules = append(cfg.TCPRules, config.TCPRule{Listen: listen, Upstream: upstream, Faults: faults})
				addDB = ask("Add another DB/TCP rule? (y/n): ")
			}

			// Write YAML
			out, err := yaml.Marshal(cfg)
			if err != nil {
				log.Fatalf("Failed to marshal YAML: %v", err)
			}
			if err := os.WriteFile(configFile, out, 0644); err != nil {
				log.Fatalf("Failed to write %s: %v", configFile, err)
			}
			log.Printf("Wrote %s with %d API rules and %d DB rules", configFile, len(cfg.Rules), len(cfg.TCPRules))
		},
	}
	configureCmd.Flags().StringVarP(&configFile, "config", "c", "faultline.yaml", "Path to the configuration file to write")
	rootCmd.AddCommand(configureCmd)

	// start: convenience command to run API and DB based on config
	var startAllCmd = &cobra.Command{
		Use:   "start",
		Short: "Start API and DB proxies based on the configuration file",
		Run: func(cmd *cobra.Command, args []string) {
			// Start API (proxy + control) and DB in parallel
			// Reuse start-api implementation by invoking the logic inline
			cfg, err := config.LoadConfig(configFile)
			if err != nil {
				log.Fatalf("Error loading config file: %v", err)
			}

			// API side using in-memory state
			ruleState := state.NewRuleState(cfg.Rules)
			var wg sync.WaitGroup
			wg.Add(1)
			go func() {
				defer wg.Done()
				p := proxy.NewProxy(ruleState)
				proxyRouter := mux.NewRouter()
				proxyRouter.PathPrefix("/").Handler(http.HandlerFunc(p.HandleRequest))
				log.Printf("-> FaultLine Proxy Server listening on port %d...", port)
				if err := http.ListenAndServe(fmt.Sprintf(":%d", port), proxyRouter); err != nil {
					log.Fatalf("Failed to start proxy server: %v", err)
				}
			}()

			// DB side
			stop := make(chan struct{})
			for _, r := range cfg.TCPRules {
				rp := tcp.NewProxy(r)
				go func(rule config.TCPRule) {
					if err := rp.Start(stop); err != nil {
						log.Printf("[DB] Proxy %s -> %s exited: %v", rule.Listen, rule.Upstream, err)
					}
				}(r)
			}
			log.Printf("[DB] Started %d DB network proxies (latency/drops/throttle/refuse). Ctrl+C to stop.", len(cfg.TCPRules))
			wg.Wait()
		},
	}
	startAllCmd.Flags().StringVarP(&configFile, "config", "c", "faultline.yaml", "Path to the configuration file")
	startAllCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port for the failure proxy server")
	rootCmd.AddCommand(startAllCmd)

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

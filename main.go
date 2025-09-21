package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"faultline/config"
	"faultline/proxy"
	"faultline/tcp"

	"github.com/spf13/cobra"
)

func main() {
	var configFile string
	var port int

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
			cfg, err := config.LoadConfig(configFile)
			if err != nil {
				log.Fatalf("Error loading config file: %v", err)
			}

			log.Printf("[API] Starting HTTP proxy on port %d", port)
			log.Printf("[API] Loaded %d API rules from %s", len(cfg.Rules), configFile)

			// Create a new proxy with the loaded configuration
			p, err := proxy.NewProxy(cfg)
			if err != nil {
				log.Fatalf("Could not create proxy: %v", err)
			}

			// The proxy will handle routing and failure injection
			http.HandleFunc("/", p.HandleRequest)

			// Start the HTTP server
			addr := fmt.Sprintf(":%d", port)
			if err := http.ListenAndServe(addr, nil); err != nil {
				log.Fatalf("Failed to start server: %v", err)
			}
		},
	}

	// Add flags to the start command
	startCmd.Flags().StringVarP(&configFile, "config", "c", "faultline.yaml", "Path to the configuration file")
	startCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port for the API proxy to listen on")

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

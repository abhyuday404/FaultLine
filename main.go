package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"faultline/config"
	"faultline/proxy"

	"github.com/spf13/cobra"
)

func main() {
	var configFile string
	var port int

	var rootCmd = &cobra.Command{
		Use:   "faultline",
		Short: "FaultLine is a tool for injecting failure scenarios into your development environment.",
		Long: `FaultLine helps you build resilient applications by simulating real-world failures 
like API latency, errors, and outages. Configure your application to use the FaultLine proxy,
define your failure scenarios in a YAML file, and start testing.`,
	}

	var startCmd = &cobra.Command{
		Use:   "start",
		Short: "Starts the FaultLine failure injection proxy",
		Run: func(cmd *cobra.Command, args []string) {
			cfg, err := config.LoadConfig(configFile)
			if err != nil {
				log.Fatalf("Error loading config file: %v", err)
			}

			log.Printf("FaultLine starting on port %d...", port)
			log.Printf("Loaded %d failure rules from %s", len(cfg.Rules), configFile)

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
	startCmd.Flags().IntVarP(&port, "port", "p", 8080, "Port for the proxy server to listen on")

	rootCmd.AddCommand(startCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

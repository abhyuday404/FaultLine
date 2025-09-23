package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

var printedBanner bool

func PrintBanner() {
	if printedBanner {
		return
	}
	if strings.TrimSpace(os.Getenv("FAULTLINE_NO_BANNER")) == "1" {
		return
	}

	blue := color.New(color.FgCyan, color.Bold)
	tip := color.New(color.FgHiBlack)
	title := color.New(color.FgWhite, color.Bold)

	banner := []string{
		"███████╗ █████╗ ██╗   ██╗██╗  ████████╗██╗     ██╗███╗   ██╗███████╗",
		"██╔════╝██╔══██╗██║   ██║██║  ╚══██╔══╝██║     ██║████╗  ██║██╔════╝",
		"█████╗  ███████║██║   ██║██║     ██║   ██║     ██║██╔██╗ ██║█████╗  ",
		"██╔══╝  ██╔══██║██║   ██║██║     ██║   ██║     ██║██║╚██╗██║██╔══╝  ",
		"██║     ██║  ██║╚██████╔╝███████╗██║   ███████╗██║██║ ╚████║███████╗",
		"╚═╝     ╚═╝  ╚═╝ ╚═════╝ ╚══════╝╚═╝   ╚══════╝╚═╝╚═╝  ╚═══╝╚══════╝",
	}

	fmt.Println()
	for _, line := range banner {
		blue.Println(line)
	}

	fmt.Println()
	title.Println("> FaultLine — Failure injection for local dev")
	tip.Println("\nTips:")
	tip.Println("  1. faultline start            # Run proxy (8080) and control API (8081)")
	tip.Println("  2. faultline rules add        # Create a latency/error/timeout rule")
	tip.Println("  3. faultline rules list       # View and toggle rules")
	tip.Println("  4. faultline endpoints list   # Discover endpoints from OpenAPI specs")
	tip.Println("  5. Use --help on any command  # More options and examples")
	fmt.Println()

	printedBanner = true
}

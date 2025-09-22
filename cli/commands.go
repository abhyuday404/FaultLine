package cli

import (
	"encoding/json"
	"faultline/codeanalysis"
	"faultline/openapi"
	"faultline/state"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/AlecAivazis/survey/v2"
	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/olekukonko/tablewriter"
	"github.com/spf13/cobra"
)

// CLI colors and styles
var (
	successColor = color.New(color.FgGreen, color.Bold)
	errorColor   = color.New(color.FgRed, color.Bold)
	warningColor = color.New(color.FgYellow, color.Bold)
	infoColor    = color.New(color.FgCyan, color.Bold)
	headerColor  = color.New(color.FgMagenta, color.Bold)
	subtleColor  = color.New(color.FgHiBlack)
)

// RuleManager handles rule operations with shared state
type RuleManager struct {
	ruleState *state.RuleState
}

// NewRuleManager creates a new rule manager that shares state with the server
func NewRuleManager(ruleState *state.RuleState) *RuleManager {
	return &RuleManager{
		ruleState: ruleState,
	}
}

// getRuleByNumber returns a rule by its display number (1-based)
func (rm *RuleManager) getRuleByNumber(number int) (*state.Rule, bool) {
	rules := rm.ruleState.GetRules()
	if number < 1 || number > len(rules) {
		return nil, false
	}
	rule := rules[number-1]
	return &rule, true
}

// GetRuleState returns the rule state for server integration
func (rm *RuleManager) GetRuleState() *state.RuleState {
	return rm.ruleState
}

// CreateCLICommands creates all CLI commands for rule management
func CreateCLICommands(rm *RuleManager) []*cobra.Command {
	var commands []*cobra.Command

	// Rules command group
	rulesCmd := &cobra.Command{
		Use:   "rules",
		Short: "Manage failure injection rules",
		Long: headerColor.Sprint(`
╔══════════════════════════════════════════════════════════════╗
║                    🚨 FaultLine Rules Manager                ║
║                                                              ║
║  Manage failure injection rules for your development         ║
║  environment with CLI commands.                              ║
╚══════════════════════════════════════════════════════════════╝
`),
	}

	// Add rule command
	addCmd := &cobra.Command{
		Use:   "add",
		Short: "Add a new failure injection rule",
		Long:  "Add a new failure injection rule with interactive prompts or flags",
		Run: func(cmd *cobra.Command, args []string) {
			addRuleInteractive(rm)
		},
	}

	// List rules command
	listCmd := &cobra.Command{
		Use:     "list",
		Short:   "List all failure injection rules",
		Aliases: []string{"ls", "show"},
		Run: func(cmd *cobra.Command, args []string) {
			listRules(rm)
		},
	}

	// Delete rule command
	deleteCmd := &cobra.Command{
		Use:     "delete [rule-id]",
		Short:   "Delete a failure injection rule",
		Aliases: []string{"del", "rm", "remove"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				deleteRuleInteractive(rm)
			} else {
				deleteRule(rm, args[0])
			}
		},
	}

	// Enable rule command
	enableCmd := &cobra.Command{
		Use:   "enable [rule-number]",
		Short: "Enable a failure injection rule by number",
		Long:  "Enable a failure injection rule using its number from the list (e.g., 'faultline rules enable 1')",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				toggleRuleInteractive(rm, true)
			} else {
				if num, err := strconv.Atoi(args[0]); err == nil {
					toggleRuleByNumber(rm, num, true)
				} else {
					errorColor.Printf("❌ Invalid rule number: %s\n", args[0])
				}
			}
		},
	}

	// Disable rule command
	disableCmd := &cobra.Command{
		Use:   "disable [rule-number]",
		Short: "Disable a failure injection rule by number",
		Long:  "Disable a failure injection rule using its number from the list (e.g., 'faultline rules disable 1')",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				toggleRuleInteractive(rm, false)
			} else {
				if num, err := strconv.Atoi(args[0]); err == nil {
					toggleRuleByNumber(rm, num, false)
				} else {
					errorColor.Printf("❌ Invalid rule number: %s\n", args[0])
				}
			}
		},
	}

	// Export rules command
	exportCmd := &cobra.Command{
		Use:   "export [filename]",
		Short: "Export rules to a JSON file",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filename := "faultline-rules.json"
			if len(args) > 0 {
				filename = args[0]
			}
			exportRules(rm, filename)
		},
	}

	// Import rules command
	importCmd := &cobra.Command{
		Use:   "import [filename]",
		Short: "Import rules from a JSON file",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) == 0 {
				errorColor.Println("❌ Please specify a filename to import from")
				return
			}
			importRules(rm, args[0])
		},
	}

	// Status command
	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show rules status and statistics",
		Run: func(cmd *cobra.Command, args []string) {
			showStatus(rm)
		},
	}

	// Add subcommands to rules command
	rulesCmd.AddCommand(addCmd, listCmd, deleteCmd, enableCmd, disableCmd, exportCmd, importCmd, statusCmd)
	commands = append(commands, rulesCmd)

	// Quick add command (shortcut)
	quickAddCmd := &cobra.Command{
		Use:   "add-rule",
		Short: "Quick add a new failure rule (shortcut)",
		Run: func(cmd *cobra.Command, args []string) {
			addRuleInteractive(rm)
		},
	}
	commands = append(commands, quickAddCmd)

	// OpenAPI Endpoints command group
	endpointsCmd := &cobra.Command{
		Use:   "endpoints",
		Short: "Discover and manage API endpoints from OpenAPI specs",
		Long: headerColor.Sprint(`
╔══════════════════════════════════════════════════════════════╗
║                  🔍 FaultLine Endpoints Discovery            ║
║                                                              ║
║  Discover API endpoints from OpenAPI/Swagger specifications  ║
║  and create failure rules automatically.                     ║
╚══════════════════════════════════════════════════════════════╝
`),
	}

	// List discovered endpoints
	listEndpointsCmd := &cobra.Command{
		Use:     "list [spec-file]",
		Short:   "List endpoints from OpenAPI specifications",
		Aliases: []string{"ls", "show"},
		Args:    cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			specFile := ""
			if len(args) > 0 {
				specFile = args[0]
			}
			listEndpoints(rm, specFile)
		},
	}

	// Discover OpenAPI specs
	discoverSpecsCmd := &cobra.Command{
		Use:   "discover [directory]",
		Short: "Discover OpenAPI specification files",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			directory := "."
			if len(args) > 0 {
				directory = args[0]
			}
			discoverSpecs(directory)
		},
	}

	// Create rules from endpoints
	createRulesCmd := &cobra.Command{
		Use:   "create-rules [spec-file]",
		Short: "Create failure rules from discovered endpoints",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			specFile := ""
			if len(args) > 0 {
				specFile = args[0]
			}
			createRulesFromEndpoints(rm, specFile)
		},
	}

	// Analyze source code for endpoints
	analyzeCodeCmd := &cobra.Command{
		Use:   "analyze-code [directory]",
		Short: "Analyze source code to find actual API endpoints being used",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			directory := "./showcase-app"
			if len(args) > 0 {
				directory = args[0]
			}
			analyzeCodeEndpoints(directory)
		},
	}

	// Compare OpenAPI specs with actual code usage
	compareCmd := &cobra.Command{
		Use:   "compare [directory]",
		Short: "Compare OpenAPI specifications with actual code usage",
		Args:  cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			directory := "./showcase-app"
			if len(args) > 0 {
				directory = args[0]
			}
			compareEndpoints(directory)
		},
	}

	// Add subcommands to endpoints command
	endpointsCmd.AddCommand(listEndpointsCmd, discoverSpecsCmd, createRulesCmd, analyzeCodeCmd, compareCmd)
	commands = append(commands, endpointsCmd)

	return commands
}

// addRuleInteractive adds a rule with interactive prompts
func addRuleInteractive(rm *RuleManager) {
	headerColor.Println("\n🚀 Creating a new failure injection rule...")

	rule := state.Rule{
		ID:      uuid.New().String(),
		Enabled: true,
	}

	// Target URL prompt
	targetPrompt := &survey.Input{
		Message: "Target URL or pattern:",
		Help:    "The URL pattern to match (e.g., https://api.example.com/users)",
	}
	survey.AskOne(targetPrompt, &rule.Target, survey.WithValidator(survey.Required))

	// Failure type selection
	failureType := ""
	failurePrompt := &survey.Select{
		Message: "Choose failure type:",
		Options: []string{"latency", "error", "timeout"},
		Help:    "latency: Add delay, error: Return HTTP error, timeout: Simulate timeout",
	}
	survey.AskOne(failurePrompt, &failureType)

	rule.Failure.Type = failureType

	// Configure failure details based on type
	switch failureType {
	case "latency":
		latencyStr := ""
		latencyPrompt := &survey.Input{
			Message: "Latency in milliseconds:",
			Default: "1000",
			Help:    "How many milliseconds to delay the response",
		}
		survey.AskOne(latencyPrompt, &latencyStr, survey.WithValidator(survey.Required))

		if latency, err := strconv.Atoi(latencyStr); err == nil {
			rule.Failure.LatencyMs = latency
		}

	case "error":
		errorCodeStr := ""
		errorPrompt := &survey.Input{
			Message: "HTTP error code:",
			Default: "500",
			Help:    "HTTP status code to return (e.g., 404, 500, 503)",
		}
		survey.AskOne(errorPrompt, &errorCodeStr, survey.WithValidator(survey.Required))

		if errorCode, err := strconv.Atoi(errorCodeStr); err == nil {
			rule.Failure.ErrorCode = errorCode
		}

	case "timeout":
		// Timeout doesn't need additional configuration
		rule.Failure.LatencyMs = 30000 // Default 30 second timeout
	}

	// Enable by default confirmation
	enabled := true
	enablePrompt := &survey.Confirm{
		Message: "Enable this rule immediately?",
		Default: true,
	}
	survey.AskOne(enablePrompt, &enabled)
	rule.Enabled = enabled

	// Add the rule
	rm.ruleState.AddRule(rule)

	// Success message
	successColor.Println("\n✅ Rule created successfully!")
	infoColor.Printf("   ID: %s\n", rule.ID)
	infoColor.Printf("   Target: %s\n", rule.Target)
	infoColor.Printf("   Type: %s\n", rule.Failure.Type)
	if rule.Failure.LatencyMs > 0 {
		infoColor.Printf("   Latency: %dms\n", rule.Failure.LatencyMs)
	}
	if rule.Failure.ErrorCode > 0 {
		infoColor.Printf("   Error Code: %d\n", rule.Failure.ErrorCode)
	}
	if rule.Enabled {
		successColor.Println("   Status: ENABLED")
	} else {
		warningColor.Println("   Status: DISABLED")
	}
}

// listRules displays all rules in a beautiful table
func listRules(rm *RuleManager) {
	rules := rm.ruleState.GetRules()

	if len(rules) == 0 {
		infoColor.Println("📝 No rules configured yet. Use 'faultline rules add' to create one!")
		return
	}

	headerColor.Printf("\n🔍 Found %d rule(s):\n\n", len(rules))

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("#", "Target", "Type", "Details", "Status")

	for i, rule := range rules {
		ruleNum := fmt.Sprintf("%d", i+1)

		target := rule.Target
		if len(target) > 40 {
			target = target[:37] + "..."
		}

		details := ""
		switch rule.Failure.Type {
		case "latency":
			details = fmt.Sprintf("%dms delay", rule.Failure.LatencyMs)
		case "error":
			details = fmt.Sprintf("HTTP %d", rule.Failure.ErrorCode)
		case "timeout":
			details = "Timeout"
		}

		status := "🔴 DISABLED"
		if rule.Enabled {
			status = "🟢 ENABLED"
		}

		table.Append(ruleNum, target, rule.Failure.Type, details, status)
	}

	table.Render()

	// Show tip for enabling/disabling
	fmt.Println()
	subtleColor.Println("💡 Tip: Use 'faultline rules enable <number>' or 'faultline rules disable <number>'")
	subtleColor.Println("   Example: faultline rules enable 1")
	fmt.Println()
} // deleteRuleInteractive deletes a rule with interactive selection
func deleteRuleInteractive(rm *RuleManager) {
	rules := rm.ruleState.GetRules()

	if len(rules) == 0 {
		warningColor.Println("⚠️  No rules found to delete")
		return
	}

	var options []string
	var ruleMap = make(map[string]state.Rule)

	for _, rule := range rules {
		option := fmt.Sprintf("%s - %s (%s)", rule.ID[:8], rule.Target, rule.Failure.Type)
		options = append(options, option)
		ruleMap[option] = rule
	}

	var selected string
	prompt := &survey.Select{
		Message: "Select rule to delete:",
		Options: options,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		return
	}

	rule := ruleMap[selected]

	// Confirmation
	confirm := false
	confirmPrompt := &survey.Confirm{
		Message: fmt.Sprintf("Are you sure you want to delete rule '%s'?", rule.Target),
		Default: false,
	}

	if err := survey.AskOne(confirmPrompt, &confirm); err != nil || !confirm {
		infoColor.Println("❌ Deletion cancelled")
		return
	}

	deleteRule(rm, rule.ID)
}

// deleteRule deletes a rule by ID
func deleteRule(rm *RuleManager, id string) {
	if rm.ruleState.DeleteRule(id) {
		successColor.Printf("✅ Rule '%s' deleted successfully\n", id)
	} else {
		errorColor.Printf("❌ Rule '%s' not found\n", id)
	}
}

// toggleRuleInteractive enables/disables a rule with interactive selection
func toggleRuleInteractive(rm *RuleManager, enable bool) {
	rules := rm.ruleState.GetRules()

	if len(rules) == 0 {
		warningColor.Println("⚠️  No rules found")
		return
	}

	action := "enable"
	if !enable {
		action = "disable"
	}

	var options []string
	var ruleNumbers []int

	for i, rule := range rules {
		if rule.Enabled == enable {
			continue // Skip already enabled/disabled rules
		}
		ruleNum := i + 1
		option := fmt.Sprintf("%d - %s (%s)", ruleNum, rule.Target, rule.Failure.Type)
		options = append(options, option)
		ruleNumbers = append(ruleNumbers, ruleNum)
	}

	if len(options) == 0 {
		infoColor.Printf("ℹ️  No rules available to %s\n", action)
		return
	}

	var selected string
	prompt := &survey.Select{
		Message: fmt.Sprintf("Select rule to %s:", action),
		Options: options,
	}

	if err := survey.AskOne(prompt, &selected); err != nil {
		return
	}

	// Extract the rule number from the selected option
	for i, option := range options {
		if option == selected {
			toggleRuleByNumber(rm, ruleNumbers[i], enable)
			return
		}
	}
}

// toggleRuleByNumber enables or disables a rule by its number
func toggleRuleByNumber(rm *RuleManager, number int, enable bool) {
	rule, exists := rm.getRuleByNumber(number)
	if !exists {
		errorColor.Printf("❌ Rule number %d not found. Use 'faultline rules list' to see available rules.\n", number)
		return
	}

	if rule.Enabled == enable {
		action := "enabled"
		if !enable {
			action = "disabled"
		}
		warningColor.Printf("⚠️  Rule %d is already %s\n", number, action)
		return
	}

	// Update the rule (auto-saves due to state persistence)
	rule.Enabled = enable
	rm.ruleState.UpdateRule(*rule)

	action := "enabled"
	emoji := "🟢"
	if !enable {
		action = "disabled"
		emoji = "🔴"
	}

	successColor.Printf("✅ Rule %d %s successfully!\n", number, action)
	infoColor.Printf("   %s %s (%s)\n", emoji, rule.Target, rule.Failure.Type)
} // exportRules exports rules to a JSON file
func exportRules(rm *RuleManager, filename string) {
	rules := rm.ruleState.GetRules()

	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		errorColor.Printf("❌ Failed to marshal rules: %v\n", err)
		return
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		errorColor.Printf("❌ Failed to write file: %v\n", err)
		return
	}

	successColor.Printf("✅ Exported %d rule(s) to '%s'\n", len(rules), filename)
}

// importRules imports rules from a JSON file
func importRules(rm *RuleManager, filename string) {
	data, err := os.ReadFile(filename)
	if err != nil {
		errorColor.Printf("❌ Failed to read file: %v\n", err)
		return
	}

	var rules []state.Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		errorColor.Printf("❌ Failed to parse JSON: %v\n", err)
		return
	}

	imported := 0
	for _, rule := range rules {
		// Generate new ID to avoid conflicts
		rule.ID = uuid.New().String()
		rm.ruleState.AddRule(rule)
		imported++
	}

	successColor.Printf("✅ Imported %d rule(s) from '%s'\n", imported, filename)
}

// showStatus displays rules status and statistics
func showStatus(rm *RuleManager) {
	rules := rm.ruleState.GetRules()

	enabled := 0
	disabled := 0
	byType := make(map[string]int)

	for _, rule := range rules {
		if rule.Enabled {
			enabled++
		} else {
			disabled++
		}
		byType[rule.Failure.Type]++
	}

	headerColor.Println("\n📊 FaultLine Rules Status")
	fmt.Println(strings.Repeat("=", 40))

	infoColor.Printf("Total Rules: %d\n", len(rules))
	if enabled > 0 {
		successColor.Printf("Enabled: %d\n", enabled)
	} else {
		subtleColor.Printf("Enabled: %d\n", enabled)
	}

	if disabled > 0 {
		warningColor.Printf("Disabled: %d\n", disabled)
	} else {
		subtleColor.Printf("Disabled: %d\n", disabled)
	}

	fmt.Println()
	infoColor.Println("By Type:")
	for failureType, count := range byType {
		infoColor.Printf("  %s: %d\n", failureType, count)
	}

	fmt.Println()
	subtleColor.Printf("Last updated: %s\n", time.Now().Format("2006-01-02 15:04:05"))
	fmt.Println()
}

// listEndpoints lists endpoints from OpenAPI specifications
func listEndpoints(rm *RuleManager, specFile string) {
	headerColor.Println("\n🔍 Discovering API Endpoints...")

	var allEndpoints []openapi.Endpoint
	var specSources []string

	if specFile != "" {
		// Parse specific spec file
		if !openapi.ValidateOpenAPIFile(specFile) {
			errorColor.Printf("❌ Invalid OpenAPI specification file: %s\n", specFile)
			return
		}

		discovered, err := openapi.ParseOpenAPISpec(specFile)
		if err != nil {
			errorColor.Printf("❌ Failed to parse OpenAPI spec %s: %v\n", specFile, err)
			return
		}

		allEndpoints = discovered.Endpoints
		specSources = append(specSources, discovered.Source)

		successColor.Printf("✅ Found %d endpoints in %s\n\n", len(discovered.Endpoints), filepath.Base(specFile))
		if discovered.Info.Title != "" {
			infoColor.Printf("API: %s", discovered.Info.Title)
			if discovered.Info.Version != "" {
				fmt.Printf(" (v%s)", discovered.Info.Version)
			}
			fmt.Println()
		}
	} else {
		// Discover all specs in current directory
		specs, err := openapi.FindOpenAPISpecs(".")
		if err != nil {
			errorColor.Printf("❌ Failed to discover OpenAPI specs: %v\n", err)
			return
		}

		if len(specs) == 0 {
			warningColor.Println("⚠️  No OpenAPI specifications found in current directory")
			subtleColor.Println("   Place your swagger.yaml, openapi.json, or api.yaml files here")
			return
		}

		for _, spec := range specs {
			if !openapi.ValidateOpenAPIFile(spec) {
				warningColor.Printf("⚠️  Skipping invalid OpenAPI file: %s\n", spec)
				continue
			}

			discovered, err := openapi.ParseOpenAPISpec(spec)
			if err != nil {
				warningColor.Printf("⚠️  Failed to parse %s: %v\n", spec, err)
				continue
			}

			allEndpoints = append(allEndpoints, discovered.Endpoints...)
			specSources = append(specSources, discovered.Source)
		}

		successColor.Printf("✅ Found %d endpoints across %d specification(s)\n\n", len(allEndpoints), len(specSources))
	}

	if len(allEndpoints) == 0 {
		warningColor.Println("⚠️  No endpoints found")
		return
	}

	// Display endpoints in a table
	table := tablewriter.NewWriter(os.Stdout)
	table.Header("#", "Method", "Path", "Full URL", "Summary")

	for i, endpoint := range allEndpoints {
		fullURL := endpoint.FullURL
		if fullURL == "" && endpoint.BaseURL != "" {
			fullURL = endpoint.BaseURL + endpoint.Path
		}

		summary := endpoint.Summary
		if summary == "" {
			summary = endpoint.Description
		}
		if len(summary) > 50 {
			summary = summary[:47] + "..."
		}

		table.Append([]string{
			fmt.Sprintf("%d", i+1),
			endpoint.Method,
			endpoint.Path,
			fullURL,
			summary,
		})
	}

	table.Render()

	fmt.Println()
	infoColor.Printf("💡 Use 'faultline endpoints create-rules' to generate failure rules from these endpoints\n")
	fmt.Println()
}

// discoverSpecs discovers OpenAPI specification files
func discoverSpecs(directory string) {
	headerColor.Printf("\n🔍 Discovering OpenAPI specifications in: %s\n\n", directory)

	specs, err := openapi.FindOpenAPISpecs(directory)
	if err != nil {
		errorColor.Printf("❌ Failed to discover OpenAPI specs: %v\n", err)
		return
	}

	if len(specs) == 0 {
		warningColor.Println("⚠️  No OpenAPI specifications found")
		subtleColor.Println("   Looking for files matching: *swagger*, *openapi*, *api*, *spec*")
		subtleColor.Println("   Supported formats: .yaml, .yml, .json")
		return
	}

	successColor.Printf("✅ Found %d potential OpenAPI specification(s):\n\n", len(specs))

	table := tablewriter.NewWriter(os.Stdout)
	table.Header("#", "File", "Valid", "Title", "Version", "Endpoints")

	validCount := 0
	for i, specPath := range specs {
		fileName := filepath.Base(specPath)
		isValid := openapi.ValidateOpenAPIFile(specPath)
		validIcon := "❌"
		title := "-"
		version := "-"
		endpointCount := "-"

		if isValid {
			validIcon = "✅"
			validCount++

			// Try to get additional info
			if discovered, err := openapi.ParseOpenAPISpec(specPath); err == nil {
				if discovered.Info.Title != "" {
					title = discovered.Info.Title
					if len(title) > 30 {
						title = title[:27] + "..."
					}
				}
				if discovered.Info.Version != "" {
					version = discovered.Info.Version
				}
				endpointCount = fmt.Sprintf("%d", len(discovered.Endpoints))
			}
		}

		table.Append([]string{
			fmt.Sprintf("%d", i+1),
			fileName,
			validIcon,
			title,
			version,
			endpointCount,
		})
	}

	table.Render()

	fmt.Println()
	infoColor.Printf("💡 Found %d valid OpenAPI specification(s) out of %d total\n", validCount, len(specs))
	infoColor.Printf("💡 Use 'faultline endpoints list [spec-file]' to see endpoints\n")
	fmt.Println()
}

// createRulesFromEndpoints creates failure rules from discovered endpoints
func createRulesFromEndpoints(rm *RuleManager, specFile string) {
	headerColor.Println("\n🚀 Creating failure rules from endpoints...")

	var allEndpoints []openapi.Endpoint

	if specFile != "" {
		// Parse specific spec file
		if !openapi.ValidateOpenAPIFile(specFile) {
			errorColor.Printf("❌ Invalid OpenAPI specification file: %s\n", specFile)
			return
		}

		discovered, err := openapi.ParseOpenAPISpec(specFile)
		if err != nil {
			errorColor.Printf("❌ Failed to parse OpenAPI spec %s: %v\n", specFile, err)
			return
		}

		allEndpoints = discovered.Endpoints
		infoColor.Printf("📖 Using spec: %s (%d endpoints)\n", filepath.Base(specFile), len(allEndpoints))
	} else {
		// Interactive spec selection
		specs, err := openapi.FindOpenAPISpecs(".")
		if err != nil {
			errorColor.Printf("❌ Failed to discover OpenAPI specs: %v\n", err)
			return
		}

		if len(specs) == 0 {
			warningColor.Println("⚠️  No OpenAPI specifications found")
			return
		}

		// Filter valid specs
		var validSpecs []string
		var specTitles []string
		for _, spec := range specs {
			if openapi.ValidateOpenAPIFile(spec) {
				validSpecs = append(validSpecs, spec)

				// Get title for display
				title := filepath.Base(spec)
				if discovered, err := openapi.ParseOpenAPISpec(spec); err == nil && discovered.Info.Title != "" {
					title = fmt.Sprintf("%s (%s)", discovered.Info.Title, filepath.Base(spec))
				}
				specTitles = append(specTitles, title)
			}
		}

		if len(validSpecs) == 0 {
			errorColor.Println("❌ No valid OpenAPI specifications found")
			return
		}

		// Select spec
		var selectedSpec string
		prompt := &survey.Select{
			Message: "Select an OpenAPI specification:",
			Options: specTitles,
		}

		var selectedIndex int
		if err := survey.AskOne(prompt, &selectedIndex); err != nil {
			errorColor.Printf("❌ Selection cancelled: %v\n", err)
			return
		}

		selectedSpec = validSpecs[selectedIndex]

		// Parse selected spec
		discovered, err := openapi.ParseOpenAPISpec(selectedSpec)
		if err != nil {
			errorColor.Printf("❌ Failed to parse selected spec: %v\n", err)
			return
		}

		allEndpoints = discovered.Endpoints
	}

	if len(allEndpoints) == 0 {
		warningColor.Println("⚠️  No endpoints found to create rules from")
		return
	}

	// Interactive rule creation options
	var createAll bool
	prompt := &survey.Confirm{
		Message: fmt.Sprintf("Create failure rules for all %d endpoints?", len(allEndpoints)),
		Default: false,
	}
	survey.AskOne(prompt, &createAll)

	var endpointsToProcess []openapi.Endpoint
	if createAll {
		endpointsToProcess = allEndpoints
	} else {
		// Let user select specific endpoints
		var options []string
		for _, endpoint := range allEndpoints {
			fullURL := endpoint.FullURL
			if fullURL == "" && endpoint.BaseURL != "" {
				fullURL = endpoint.BaseURL + endpoint.Path
			}
			option := fmt.Sprintf("%s %s (%s)", endpoint.Method, endpoint.Path, fullURL)
			options = append(options, option)
		}

		var selectedIndices []int
		multiPrompt := &survey.MultiSelect{
			Message: "Select endpoints to create rules for:",
			Options: options,
		}

		if err := survey.AskOne(multiPrompt, &selectedIndices); err != nil {
			errorColor.Printf("❌ Selection cancelled: %v\n", err)
			return
		}

		for _, index := range selectedIndices {
			endpointsToProcess = append(endpointsToProcess, allEndpoints[index])
		}
	}

	if len(endpointsToProcess) == 0 {
		warningColor.Println("⚠️  No endpoints selected")
		return
	}

	// Create rules
	created := 0
	for _, endpoint := range endpointsToProcess {
		fullURL := endpoint.FullURL
		if fullURL == "" && endpoint.BaseURL != "" {
			fullURL = endpoint.BaseURL + endpoint.Path
		}

		rule := state.Rule{
			ID:      uuid.New().String(),
			Target:  fullURL,
			Enabled: false, // Start disabled by default
			Failure: state.Failure{
				Type:      "latency",
				LatencyMs: 2000,
			},
		}

		rm.ruleState.AddRule(rule)
		created++

		subtleColor.Printf("  ✓ Created rule for %s %s\n", endpoint.Method, endpoint.Path)
	}

	fmt.Println()
	successColor.Printf("✅ Created %d failure rule(s) from endpoints\n", created)
	infoColor.Println("💡 Use 'faultline rules list' to see all rules")
	infoColor.Println("💡 Enable rules with 'faultline rules enable <rule-number>'")
	fmt.Println()
}

// analyzeCodeEndpoints analyzes source code to discover actual API endpoints
func analyzeCodeEndpoints(directory string) {
	headerColor.Printf("\n🔍 Analyzing source code in: %s\n\n", directory)

	result, err := codeanalysis.AnalyzeDirectory(directory)
	if err != nil {
		errorColor.Printf("❌ Failed to analyze code: %v\n", err)
		return
	}

	if len(result.Endpoints) == 0 {
		warningColor.Println("⚠️  No API endpoints found in source code")
		return
	}

	// Display summary
	successColor.Printf("✅ Found %d endpoint(s) in %d file(s)\n\n", len(result.Endpoints), len(result.Files))

	// Display methods breakdown
	if len(result.MethodCounts) > 0 {
		subtleColor.Println("📊 HTTP Methods:")
		for method, count := range result.MethodCounts {
			fmt.Printf("  %s: %d\n", method, count)
		}
		fmt.Println()
	}

	// Display unique URLs
	if len(result.UniqueURLs) > 0 {
		infoColor.Println("🌐 Unique URLs discovered:")
		for _, url := range result.UniqueURLs {
			fmt.Printf("  %s\n", url)
		}
		fmt.Println()
	}

	// Display detailed endpoints
	headerColor.Println("📋 Detailed Endpoints:")
	for i, endpoint := range result.Endpoints {
		fmt.Printf("  %d. %s %s\n", i+1, endpoint.Method, endpoint.URL)
		if endpoint.File != "" {
			subtleColor.Printf("     📁 File: %s (line %d)\n", endpoint.File, endpoint.Line)
		}
		if endpoint.Context != "" {
			subtleColor.Printf("     📝 Context: %s\n", endpoint.Context)
		}
		fmt.Println()
	}

	// Suggest next actions
	infoColor.Println("💡 Next steps:")
	infoColor.Printf("   • Create rules: faultline endpoints create-rules-from-code %s\n", directory)
	infoColor.Printf("   • Compare with OpenAPI: faultline endpoints compare %s\n", directory)
}

// compareEndpoints compares OpenAPI specs with actual code usage
func compareEndpoints(directory string) {
	headerColor.Printf("\n🔍 Comparing OpenAPI specs with code usage in: %s\n\n", directory)

	// Analyze code endpoints
	codeResult, err := codeanalysis.AnalyzeDirectory(directory)
	if err != nil {
		errorColor.Printf("❌ Failed to analyze code: %v\n", err)
		return
	}

	// Discover OpenAPI specs
	openAPISpecs, err := openapi.FindOpenAPISpecs(".")
	if err != nil {
		errorColor.Printf("❌ Failed to discover OpenAPI specs: %v\n", err)
		return
	}

	// Parse all discovered specs
	var allOpenAPIEndpoints []openapi.Endpoint
	for _, specPath := range openAPISpecs {
		endpoints, err := openapi.ParseOpenAPISpec(specPath)
		if err != nil {
			warningColor.Printf("⚠️  Failed to parse %s: %v\n", specPath, err)
			continue
		}
		allOpenAPIEndpoints = append(allOpenAPIEndpoints, endpoints.Endpoints...)
	}

	// Display comparison results
	successColor.Printf("✅ Analysis complete\n\n")

	infoColor.Printf("📊 Summary:\n")
	fmt.Printf("  OpenAPI Endpoints: %d\n", len(allOpenAPIEndpoints))
	fmt.Printf("  Code Endpoints: %d\n", len(codeResult.Endpoints))
	fmt.Printf("  Code Files: %d\n", len(codeResult.Files))
	fmt.Println()

	// Find endpoints only in OpenAPI
	codeURLs := make(map[string]bool)
	for _, ep := range codeResult.Endpoints {
		key := fmt.Sprintf("%s %s", ep.Method, ep.URL)
		codeURLs[key] = true
	}

	openAPIURLs := make(map[string]bool)
	var onlyInOpenAPI []openapi.Endpoint
	for _, ep := range allOpenAPIEndpoints {
		key := fmt.Sprintf("%s %s", ep.Method, ep.Path)
		openAPIURLs[key] = true
		if !codeURLs[key] {
			onlyInOpenAPI = append(onlyInOpenAPI, ep)
		}
	}

	// Find endpoints only in code
	var onlyInCode []codeanalysis.EndpointUsage
	for _, ep := range codeResult.Endpoints {
		key := fmt.Sprintf("%s %s", ep.Method, ep.URL)
		if !openAPIURLs[key] {
			onlyInCode = append(onlyInCode, ep)
		}
	}

	// Display differences
	if len(onlyInOpenAPI) > 0 {
		warningColor.Printf("⚠️  Endpoints in OpenAPI but not used in code (%d):\n", len(onlyInOpenAPI))
		for _, ep := range onlyInOpenAPI {
			fmt.Printf("  %s %s\n", ep.Method, ep.Path)
		}
		fmt.Println()
	}

	if len(onlyInCode) > 0 {
		warningColor.Printf("⚠️  Endpoints used in code but not in OpenAPI (%d):\n", len(onlyInCode))
		for _, ep := range onlyInCode {
			fmt.Printf("  %s %s (%s:%d)\n", ep.Method, ep.URL, ep.File, ep.Line)
		}
		fmt.Println()
	}

	if len(onlyInOpenAPI) == 0 && len(onlyInCode) == 0 {
		successColor.Println("🎉 Perfect match! All endpoints are consistent between OpenAPI specs and code.")
	}

	// Suggest next actions
	infoColor.Println("💡 Suggested actions:")
	if len(onlyInCode) > 0 {
		infoColor.Println("   • Update OpenAPI specs to include missing endpoints")
	}
	if len(onlyInOpenAPI) > 0 {
		infoColor.Println("   • Remove unused endpoints from OpenAPI specs, or implement them in code")
	}
}

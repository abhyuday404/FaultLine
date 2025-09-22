package cli

import (
	"encoding/json"
	"faultline/state"
	"fmt"
	"os"
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

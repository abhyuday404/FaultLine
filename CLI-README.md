# FaultLine CLI - Beautiful Command Line Interface

## Overview

I've created a comprehensive and beautiful CLI for FaultLine that provides an alternative to the control panel for managing failure injection rules. The CLI offers rich formatting, interactive prompts, and intuitive commands.

## Features

### 🎨 Beautiful Interface
- Colorful output with emojis and styled text
- Professional table formatting for rule listings
- Interactive prompts with validation
- Clear success/error messaging
- Status dashboards with statistics

### 🚀 Command Structure

```bash
faultline [command] [flags]
```

#### Main Commands:

1. **`faultline start`** - Start the FaultLine proxy and API servers
2. **`faultline rules`** - Manage failure injection rules
3. **`faultline add-rule`** - Quick shortcut to add a rule

#### Rules Management Commands:

- **`faultline rules add`** - Add a new rule (interactive)
- **`faultline rules list`** - List all rules in a beautiful table
- **`faultline rules delete [rule-id]`** - Delete a rule
- **`faultline rules enable [rule-id]`** - Enable a rule
- **`faultline rules disable [rule-id]`** - Disable a rule
- **`faultline rules status`** - Show rules statistics
- **`faultline rules export [filename]`** - Export rules to JSON
- **`faultline rules import [filename]`** - Import rules from JSON

## Key Features

### 📊 Rule Types Supported
- **Latency** - Add delays to responses
- **Error** - Return HTTP error codes
- **Timeout** - Simulate request timeouts

### 💾 Persistent Storage
- Rules are automatically saved to `faultline-rules.json`
- Custom data file can be specified with `--data` flag
- Import/export functionality for rule sharing

### 🎯 Interactive Mode
- Guided prompts for rule creation
- Input validation and help text
- Dropdown selections for rule types
- Confirmation prompts for destructive operations

### 🎨 Beautiful Output
- Color-coded status indicators
- Professional table formatting
- Progress indicators and emojis
- Clear error and success messages

## Usage Examples

### Starting FaultLine
```bash
# Start with default ports
./faultline start

# Start with custom ports
./faultline start --proxy-port 9090 --api-port 9091
```

### Managing Rules
```bash
# Add a new rule interactively
./faultline rules add

# List all rules
./faultline rules list

# Show status and statistics
./faultline rules status

# Export rules to backup
./faultline rules export backup.json

# Import rules from file
./faultline rules import backup.json
```

### Quick Rule Creation
```bash
# Quick add shortcut
./faultline add-rule
```

## Integration with Existing Features

### 🔄 Server Integration
- CLI rules are automatically loaded when starting the server
- Rules persist between restarts
- Compatible with existing API endpoints

### 📝 Data Persistence
- Rules stored in JSON format
- Automatic loading on startup
- Backup and restore capabilities

### 🎛️ Control Panel Alternative
- Full feature parity with web control panel
- Scriptable and automation-friendly
- Better for CI/CD integration

## Sample Output

### Rules List
```
🔍 Found 3 rule(s):

┌─────────────┬────────────────────────────────┬─────────┬──────────────┬─────────────┐
│     ID      │             TARGET             │  TYPE   │   DETAILS    │   STATUS    │
├─────────────┼────────────────────────────────┼─────────┼──────────────┼─────────────┤
│ e45e0ef3... │ https://api.example.com/users  │ latency │ 2000ms delay │ 🟢 ENABLED  │
│ 95edc7ed... │ https://api.example.com/orders │ error   │ HTTP 500     │ 🔴 DISABLED │
│ a68d9368... │ https://api.example.com/pay... │ timeout │ Timeout      │ 🟢 ENABLED  │
└─────────────┴────────────────────────────────┴─────────┴──────────────┴─────────────┘
```

### Status Dashboard
```
📊 FaultLine Rules Status
========================================
Total Rules: 3
Enabled: 2
Disabled: 1

By Type:
  latency: 1
  error: 1
  timeout: 1

Last updated: 2025-09-22 16:03:35
```

## Dependencies Added

- **github.com/olekukonko/tablewriter** - Beautiful table formatting
- **github.com/fatih/color** - Terminal colors and styling
- **github.com/AlecAivazis/survey/v2** - Interactive prompts and forms

## Architecture

### CLI Package Structure
- `cli/commands.go` - Main CLI implementation
- Modular design with separate functions for each command
- Clean separation between CLI logic and core application
- Rule persistence layer with JSON storage

### Integration Points
- Shares the same `state.RuleState` with the server
- Rules are loaded from persistent storage on startup
- CLI commands modify the same data structures used by the API

## Benefits Over Control Panel

1. **Scriptability** - Can be automated and used in scripts
2. **Speed** - No need to open a browser or navigate web UI
3. **Accessibility** - Works in any terminal environment
4. **Version Control** - Rules can be version controlled as JSON files
5. **CI/CD Integration** - Easy to integrate into deployment pipelines
6. **Keyboard-focused** - Faster for power users
7. **Network Independence** - Works without network access to web UI

This CLI provides a complete alternative to the control panel while maintaining all the functionality and adding powerful new capabilities for rule management!
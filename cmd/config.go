package cmd

import (
	"fmt"

	"github.com/abhinavcdev/gflow/internal/config"
	"github.com/abhinavcdev/gflow/internal/ui"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "View or edit gflow configuration",
	Long: `Display the current gflow configuration or edit specific values.

Examples:
  gflow config                    # Show current config
  gflow config --path             # Show config file path
  gflow config set pr.draft true  # Set a config value`,
	RunE: runConfig,
}

var (
	configShowPath bool
)

func init() {
	configCmd.Flags().BoolVar(&configShowPath, "path", false, "show config file path")

	configCmd.AddCommand(configSetCmd)
	configCmd.AddCommand(configListStrategiesCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	if configShowPath {
		path, err := config.FindConfigFile()
		if err != nil {
			ui.Error("No config file found")
			return err
		}
		fmt.Println(path)
		return nil
	}

	ui.Title("  Configuration")
	fmt.Println()

	// Print the config as YAML
	data, err := yaml.Marshal(cfg)
	if err != nil {
		ui.Error("Failed to marshal config")
		return err
	}

	fmt.Println(ui.MutedStyle.Render(string(data)))

	path, err := config.FindConfigFile()
	if err == nil {
		ui.Detail("Config file", path)
	}
	fmt.Println()

	return nil
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a configuration value",
	Long: `Set a configuration value in .gflow.yml.

Examples:
  gflow config set pr.draft true
  gflow config set pr.merge_method squash
  gflow config set branching.main main
  gflow config set provider.name gitlab`,
	Args: cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		key := args[0]
		value := args[1]

		path, err := config.FindConfigFile()
		if err != nil {
			ui.Error("No config file found — run 'gflow init' first")
			return err
		}

		currentCfg, err := config.LoadFromFile(path)
		if err != nil {
			ui.Error("Failed to load config")
			return err
		}

		// Simple key-value setter for common settings
		switch key {
		case "provider.name":
			currentCfg.Provider.Name = value
		case "provider.owner":
			currentCfg.Provider.Owner = value
		case "provider.repo":
			currentCfg.Provider.Repo = value
		case "provider.host":
			currentCfg.Provider.Host = value
		case "provider.token_env":
			currentCfg.Provider.TokenEnv = value
		case "branching.main":
			currentCfg.Branching.Main = value
		case "branching.develop":
			currentCfg.Branching.Develop = value
		case "branching.use_develop":
			currentCfg.Branching.UseDevelop = value == "true"
		case "pr.draft":
			currentCfg.PR.Draft = value == "true"
		case "pr.auto_assign":
			currentCfg.PR.AutoAssign = value == "true"
		case "pr.merge_method":
			currentCfg.PR.MergeMethod = value
		case "pr.delete_branch_on_merge":
			currentCfg.PR.DeleteBranch = value == "true"
		case "pr.default_base":
			currentCfg.PR.DefaultBase = value
		case "commit.convention":
			currentCfg.Commit.Convention = value
		case "commit.require_scope":
			currentCfg.Commit.RequireScope = value == "true"
		case "commit.require_ticket":
			currentCfg.Commit.RequireTicket = value == "true"
		case "commit.ticket_pattern":
			currentCfg.Commit.TicketPattern = value
		default:
			ui.Errorf("Unknown config key: %s", key)
			ui.Info("Run 'gflow config' to see available keys")
			return fmt.Errorf("unknown key: %s", key)
		}

		if err := config.Save(currentCfg, path); err != nil {
			ui.Error("Failed to save config")
			return err
		}

		ui.SuccessMsg(fmt.Sprintf("Set %s = %s", key, value))
		return nil
	},
}

var configListStrategiesCmd = &cobra.Command{
	Use:   "strategies",
	Short: "List all available strategies (built-in and custom)",
	RunE: func(cmd *cobra.Command, args []string) error {
		ui.Title("  Available Strategies")
		fmt.Println()

		// Built-in strategies
		builtins := []struct {
			name   string
			prefix string
			base   string
		}{
			{"feature", cfg.Branching.Prefixes.Feature, cfg.GetBaseBranch()},
			{"bugfix", cfg.Branching.Prefixes.Bugfix, cfg.GetBaseBranch()},
			{"hotfix", cfg.Branching.Prefixes.Hotfix, cfg.Branching.Main},
			{"release", cfg.Branching.Prefixes.Release, cfg.GetBaseBranch()},
		}

		fmt.Println(ui.BoldStyle.Render("  Built-in:"))
		for _, b := range builtins {
			fmt.Printf("    %s %s %s %s\n",
				ui.SuccessStyle.Render(ui.IconDot),
				ui.BoldStyle.Render(fmt.Sprintf("%-12s", b.name)),
				ui.MutedStyle.Render(fmt.Sprintf("prefix=%s", b.prefix)),
				ui.MutedStyle.Render(fmt.Sprintf("base=%s", b.base)),
			)
		}

		if len(cfg.Strategies) > 0 {
			fmt.Println()
			fmt.Println(ui.BoldStyle.Render("  Custom:"))
			for name, s := range cfg.Strategies {
				desc := s.Description
				if desc == "" {
					desc = "no description"
				}
				fmt.Printf("    %s %s %s\n",
					ui.SuccessStyle.Render(ui.IconStar),
					ui.BoldStyle.Render(fmt.Sprintf("%-12s", name)),
					ui.MutedStyle.Render(desc),
				)
				if s.Prefix != "" {
					fmt.Printf("      %s %s\n", ui.MutedStyle.Render("prefix:"), s.Prefix)
				}
				if s.BaseBranch != "" {
					fmt.Printf("      %s %s\n", ui.MutedStyle.Render("base:"), s.BaseBranch)
				}
			}
		}

		fmt.Println()
		ui.Info("Usage: gflow start <strategy> <name>")
		fmt.Println()

		return nil
	},
}

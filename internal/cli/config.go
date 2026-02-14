package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/lieyanc/fire-commit/internal/config"
	"github.com/lieyanc/fire-commit/internal/tui/setup"
	"gopkg.in/yaml.v3"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Open interactive configuration editor",
	RunE:  runConfigEdit,
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Display current configuration as YAML",
	RunE:  runConfigShow,
}

var configSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Re-run the setup wizard",
	RunE:  runConfigSetup,
}

func init() {
	configCmd.AddCommand(configShowCmd)
	configCmd.AddCommand(configSetupCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigEdit(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		fmt.Println("No configuration found. Running setup wizard...")
		_, err := setup.RunWizard()
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	_, err = setup.RunConfigEditor(cfg)
	return err
}

func runConfigShow(cmd *cobra.Command, args []string) error {
	if !config.Exists() {
		fmt.Println("No configuration found. Run 'firecommit config setup' to create one.")
		return nil
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Mask API keys for display
	display := *cfg
	display.Providers = make(map[string]config.ProviderConfig)
	for name, p := range cfg.Providers {
		masked := p
		if len(p.APIKey) > 8 {
			masked.APIKey = p.APIKey[:4] + "..." + p.APIKey[len(p.APIKey)-4:]
		} else if p.APIKey != "" {
			masked.APIKey = "****"
		}
		display.Providers[name] = masked
	}

	data, err := yaml.Marshal(&display)
	if err != nil {
		return err
	}

	fmt.Printf("Config file: %s\n\n", config.ConfigPath())
	os.Stdout.Write(data)
	return nil
}

func runConfigSetup(cmd *cobra.Command, args []string) error {
	_, err := setup.RunWizard()
	return err
}

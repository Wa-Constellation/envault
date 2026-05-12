package cmd

import (
	"fmt"
	"os"

	"github.com/envault/envault/internal/config"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Print or set configuration",
	Long: `Without flags: print current configuration (backend, config path, vault path).

With --set-backend: set the default backend.
  envault config --set-backend <keyring|file>`,
	RunE: runConfig,
}

func init() {
	configCmd.Flags().String("set-backend", "", "Set the default backend (keyring|file)")
	rootCmd.AddCommand(configCmd)
}

func runConfig(cmd *cobra.Command, args []string) error {
	setBackend, _ := cmd.Flags().GetString("set-backend")

	configPath := cfgFile
	if configPath == "" {
		configPath = config.DefaultConfigPath()
	}

	if setBackend != "" {
		// Set backend
		if setBackend != "keyring" && setBackend != "file" {
			return fmt.Errorf("invalid backend %q (use 'keyring' or 'file')", setBackend)
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			return err
		}

		cfg.Backend = setBackend
		if err := cfg.Save(configPath); err != nil {
			return err
		}

		fmt.Fprintf(os.Stderr, "Default backend set to %q\n", setBackend)
		return nil
	}

	// Print current config
	cfg, err := config.Load(configPath)
	if err != nil {
		return err
	}

	fmt.Printf("Config file:  %s\n", configPath)
	fmt.Printf("Backend:      %s\n", cfg.Backend)
	fmt.Printf("Vault file:   %s\n", cfg.VaultFile)
	fmt.Printf("Default TTL:  %s\n", cfg.DefaultTTL)

	return nil
}

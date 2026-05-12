package cmd

import (
	"fmt"
	"os"

	"github.com/Wa-Constellation/envault/internal/backend"
	"github.com/Wa-Constellation/envault/internal/config"
	"github.com/Wa-Constellation/envault/internal/profile"
	"github.com/spf13/cobra"
)

var (
	version     = "dev"
	cfgFile     string
	vaultFile   string
	backendFlag string
	yesFlag     bool
	verbose     bool
)

// SetVersion sets the version string (called from main).
func SetVersion(v string) {
	version = v
}

var rootCmd = &cobra.Command{
	Use:   "envault",
	Short: "Manage encrypted environment variables",
	Long: `envault organizes environment variables into profiles.
Profiles are stored encrypted and injected into subprocesses on demand.
Secrets never touch disk in plaintext and only live in the child process's memory.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Version:       version,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&backendFlag, "backend", "", "Override backend (keyring|file)")
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "Override config file path")
	rootCmd.PersistentFlags().StringVar(&vaultFile, "vault-file", "", "Override encrypted file path (file backend only)")
	rootCmd.PersistentFlags().BoolVar(&yesFlag, "yes", false, "Skip confirmation prompts")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Verbose output for debugging")
}

// loadConfig loads the configuration, respecting flag overrides.
func loadConfig() (*config.Config, error) {
	cfg, err := config.Load(cfgFile)
	if err != nil {
		return nil, err
	}

	// Apply flag overrides
	if backendFlag != "" {
		cfg.Backend = backendFlag
	}
	if vaultFile != "" {
		cfg.VaultFile = vaultFile
	}

	return cfg, nil
}

// getStore creates a profile store from the current configuration.
func getStore() (*profile.Store, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	var b backend.Backend
	switch cfg.Backend {
	case "file", "":
		b = backend.NewFileBackend(cfg.VaultFile)
	case "keyring":
		b = backend.NewKeyringBackend()
	default:
		return nil, fmt.Errorf("unknown backend %q (use 'file' or 'keyring')", cfg.Backend)
	}

	return profile.NewStore(b), nil
}

// verboseLog prints a message if verbose mode is enabled.
func verboseLog(format string, args ...any) {
	if verbose {
		fmt.Fprintf(os.Stderr, "[verbose] "+format+"\n", args...)
	}
}

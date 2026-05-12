package cmd

import (
	"fmt"
	"os"

	"github.com/Wa-Constellation/envault/internal/backend"
	"github.com/Wa-Constellation/envault/internal/profile"
	"github.com/spf13/cobra"
)

var migrateCmd = &cobra.Command{
	Use:   "migrate --from <backend> --to <backend>",
	Short: "Migrate profiles between backends",
	Long: `Copy all profiles from one backend to another.
Profiles are read into memory and written to the destination — secrets never touch disk during migration.

Examples:
  envault migrate --from file --to keyring
  envault migrate --from keyring --to file`,
	RunE: runMigrate,
}

func init() {
	migrateCmd.Flags().String("from", "", "Source backend (file|keyring)")
	migrateCmd.Flags().String("to", "", "Destination backend (file|keyring)")
	_ = migrateCmd.MarkFlagRequired("from")
	_ = migrateCmd.MarkFlagRequired("to")
	rootCmd.AddCommand(migrateCmd)
}

func resolveBackend(name string) (backend.Backend, error) {
	switch name {
	case "file":
		cfg, err := loadConfig()
		if err != nil {
			return nil, err
		}
		return backend.NewFileBackend(cfg.VaultFile), nil
	case "keyring":
		return backend.NewKeyringBackend(), nil
	default:
		return nil, fmt.Errorf("unknown backend %q (use 'file' or 'keyring')", name)
	}
}

func runMigrate(cmd *cobra.Command, args []string) error {
	fromName, _ := cmd.Flags().GetString("from")
	toName, _ := cmd.Flags().GetString("to")

	if fromName == toName {
		return fmt.Errorf("source and destination backends are the same (%q)", fromName)
	}

	srcBackend, err := resolveBackend(fromName)
	if err != nil {
		return fmt.Errorf("source: %w", err)
	}

	dstBackend, err := resolveBackend(toName)
	if err != nil {
		return fmt.Errorf("destination: %w", err)
	}

	src := profile.NewStore(srcBackend)
	dst := profile.NewStore(dstBackend)

	names, err := src.List()
	if err != nil {
		return fmt.Errorf("listing source profiles: %w", err)
	}

	if len(names) == 0 {
		fmt.Fprintf(os.Stderr, "No profiles found in %s backend, nothing to migrate.\n", fromName)
		return nil
	}

	fmt.Fprintf(os.Stderr, "Migrating %d profile(s) from %s to %s...\n", len(names), fromName, toName)

	for _, name := range names {
		p, err := src.Get(name)
		if err != nil {
			return fmt.Errorf("reading profile %q: %w", name, err)
		}
		if err := dst.Put(p); err != nil {
			return fmt.Errorf("writing profile %q: %w", name, err)
		}
		fmt.Fprintf(os.Stderr, "  %s\n", name)
	}

	fmt.Fprintf(os.Stderr, "Done. %d profile(s) migrated to %s.\n", len(names), toName)
	fmt.Fprintf(os.Stderr, "Run 'envault config --set-backend %s' to switch your default backend.\n", toName)

	return nil
}

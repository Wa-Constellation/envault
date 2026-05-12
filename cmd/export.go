package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/envault/envault/internal/awssts"
	"github.com/envault/envault/internal/dotenv"
	"github.com/spf13/cobra"
)

var exportCmd = &cobra.Command{
	Use:   "export <profile> [--format <format>]",
	Short: "Print profile vars to stdout",
	Long: `Print the profile's vars to stdout in the specified format.
Formats:
  --format shell    (default)  export KEY="VALUE"
  --format dotenv              KEY=VALUE
  --format json                {"KEY": "VALUE", ...}

WARNING: this prints secrets to stdout. Use 'exec' when possible.`,
	Args: cobra.ExactArgs(1),
	RunE: runExport,
}

func init() {
	exportCmd.Flags().String("format", "shell", "Output format: shell, dotenv, json")
	exportCmd.Flags().Bool("no-sts", false, "Skip automatic STS temporary credential generation for AWS keys")
	rootCmd.AddCommand(exportCmd)
}

func runExport(cmd *cobra.Command, args []string) error {
	profileName := args[0]

	store, err := getStore()
	if err != nil {
		return err
	}

	p, err := store.Get(profileName)
	if err != nil {
		return fmt.Errorf("profile %q: %w", profileName, err)
	}

	// Check TTL expiry
	if p.IsExpired() {
		return fmt.Errorf(
			"profile %q expired %s ago (TTL: %s, last updated: %s)\n"+
				"Run 'envault add %s ...' to refresh",
			profileName,
			time.Since(p.UpdatedAt).Round(time.Second),
			time.Duration(p.TTL),
			p.UpdatedAt.Format(time.RFC3339),
			profileName,
		)
	}

	// Resolve inheritance
	vars, err := p.Resolve(store)
	if err != nil {
		return err
	}

	// If profile has AWS_ROLE_ARN + credentials, assume the role via STS
	noSTS, _ := cmd.Flags().GetBool("no-sts")
	if !noSTS && awssts.ShouldAssumeRole(vars) {
		if err := awssts.AssumeRole(vars); err != nil {
			return fmt.Errorf("AWS AssumeRole failed: %w\n"+
				"Use --no-sts to skip", err)
		}
	}

	format, _ := cmd.Flags().GetString("format")

	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	switch format {
	case "shell", "":
		for _, k := range keys {
			fmt.Printf("export %s=%q\n", k, vars[k])
		}
	case "dotenv":
		for _, k := range keys {
			fmt.Printf("%s=%s\n", k, dotenv.Quote(vars[k]))
		}
	case "json":
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(vars)
	default:
		return fmt.Errorf("unknown format %q (use shell, dotenv, or json)", format)
	}

	return nil
}

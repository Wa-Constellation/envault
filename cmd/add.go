package cmd

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/Wa-Constellation/envault/internal/backend"
	"github.com/Wa-Constellation/envault/internal/dotenv"
	"github.com/Wa-Constellation/envault/internal/profile"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var addCmd = &cobra.Command{
	Use:   "add <profile> [KEY=VALUE ...] [flags]",
	Short: "Store environment variables into a profile",
	Long: `Store one or more env vars into a profile.
Creates the profile if it doesn't exist. Overwrites existing keys silently.

Three modes:
  envault add <profile> KEY=VALUE [KEY2=VALUE2 ...]
  envault add <profile> --from-env KEY [KEY2 ...]
  envault add <profile> --from-file <path>

If a KEY is provided without =VALUE, you will be prompted to enter the value
interactively (value won't appear in shell history).`,
	Args: cobra.MinimumNArgs(1),
	RunE: runAdd,
}

func init() {
	addCmd.Flags().StringSlice("from-env", nil, "Capture env var values from the calling environment")
	addCmd.Flags().String("from-file", "", "Import vars from a dotenv-formatted file")
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	profileName := args[0]

	if err := profile.ValidateProfileName(profileName); err != nil {
		return err
	}

	store, err := getStore()
	if err != nil {
		return err
	}

	// Load existing profile or create new
	p, err := store.Get(profileName)
	if err != nil {
		if errors.Is(err, backend.ErrProfileNotFound) {
			p = &profile.Profile{
				Name:      profileName,
				Vars:      make(map[string]string),
				CreatedAt: time.Now(),
			}
			verboseLog("creating new profile %q", profileName)
		} else {
			return err
		}
	}

	fromEnv, _ := cmd.Flags().GetStringSlice("from-env")
	fromFile, _ := cmd.Flags().GetString("from-file")

	switch {
	case fromFile != "":
		vars, err := dotenv.ParseFile(fromFile)
		if err != nil {
			return fmt.Errorf("parsing dotenv file: %w", err)
		}
		for k, v := range vars {
			if err := profile.ValidateEnvKey(k); err != nil {
				return err
			}
			p.Vars[k] = v
		}
		fmt.Fprintf(os.Stderr, "Imported %d vars from %s into profile %q\n", len(vars), fromFile, profileName)

	case len(fromEnv) > 0:
		for _, key := range fromEnv {
			if err := profile.ValidateEnvKey(key); err != nil {
				return err
			}
			val, ok := os.LookupEnv(key)
			if !ok {
				return fmt.Errorf("env var %q is not set in the current environment", key)
			}
			p.Vars[key] = val
		}
		fmt.Fprintf(os.Stderr, "Captured %d vars from environment into profile %q\n", len(fromEnv), profileName)

	default:
		if len(args) < 2 {
			return fmt.Errorf("usage: envault add <profile> KEY=VALUE [KEY2=VALUE2 ...]\n" +
				"       envault add <profile> --from-env KEY [KEY2 ...]\n" +
				"       envault add <profile> --from-file <path>")
		}

		for _, arg := range args[1:] {
			parts := strings.SplitN(arg, "=", 2)
			key := parts[0]

			if err := profile.ValidateEnvKey(key); err != nil {
				return err
			}

			if len(parts) == 2 {
				// KEY=VALUE form
				p.Vars[key] = parts[1]
			} else {
				// KEY only — prompt interactively
				fmt.Fprintf(os.Stderr, "Enter value for %s: ", key)
				value, err := term.ReadPassword(int(syscall.Stdin)) //nolint:unconvert // syscall.Stdin is uintptr on Windows
				fmt.Fprintln(os.Stderr)                             // newline after password input
				if err != nil {
					return fmt.Errorf("reading value for %s: %w", key, err)
				}
				p.Vars[key] = string(value)
			}
		}
		fmt.Fprintf(os.Stderr, "Added %d vars to profile %q\n", len(args)-1, profileName)
	}

	p.UpdatedAt = time.Now()
	return store.Put(p)
}

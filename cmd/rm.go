package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Wa-Constellation/envault/internal/profile"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <profile> [KEY ...]",
	Short: "Remove a profile or specific keys from a profile",
	Long: `Without key arguments: remove an entire profile and all its vars.
Asks for confirmation unless --yes is passed.

With key arguments: remove specific keys from a profile.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runRm,
}

func init() {
	rootCmd.AddCommand(rmCmd)
}

func runRm(cmd *cobra.Command, args []string) error {
	profileName := args[0]

	store, err := getStore()
	if err != nil {
		return err
	}

	if len(args) == 1 {
		// Remove entire profile
		if !yesFlag {
			if !confirmPrompt(fmt.Sprintf("Remove profile %q and all its vars?", profileName)) {
				fmt.Fprintln(os.Stderr, "Cancelled.")
				return nil
			}
		}

		if err := store.Delete(profileName); err != nil {
			return fmt.Errorf("removing profile %q: %w", profileName, err)
		}
		fmt.Fprintf(os.Stderr, "Removed profile %q\n", profileName)
		return nil
	}

	// Remove specific keys
	p, err := store.Get(profileName)
	if err != nil {
		return fmt.Errorf("profile %q: %w", profileName, err)
	}

	removed := 0
	for _, key := range args[1:] {
		if err := profile.ValidateEnvKey(key); err != nil {
			return err
		}
		if _, exists := p.Vars[key]; exists {
			delete(p.Vars, key)
			removed++
		} else {
			fmt.Fprintf(os.Stderr, "Warning: key %q not found in profile %q\n", key, profileName)
		}
	}

	if removed > 0 {
		p.UpdatedAt = time.Now()
		if err := store.Put(p); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Removed %d key(s) from profile %q\n", removed, profileName)
	}

	return nil
}

// confirmPrompt asks the user for confirmation (y/N).
func confirmPrompt(question string) bool {
	fmt.Fprintf(os.Stderr, "%s [y/N] ", question)
	reader := bufio.NewReader(os.Stdin)
	answer, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

package cmd

import (
	"fmt"
	"os"

	"github.com/envault/envault/internal/profile"
	"github.com/spf13/cobra"
)

var setTTLCmd = &cobra.Command{
	Use:   "set-ttl <profile> <duration>",
	Short: "Set or update the TTL for a profile",
	Long: `Set or update the TTL for a profile.
Duration format: "1h", "30m", "24h", "7d", "0" (disable).
TTL countdown starts from the last 'envault add' operation on the profile.`,
	Args: cobra.ExactArgs(2),
	RunE: runSetTTL,
}

var setInheritCmd = &cobra.Command{
	Use:   "set-inherit <profile> <parent-profile>",
	Short: "Set profile inheritance",
	Long: `Set profile to inherit vars from parent-profile.
Inheritance is single-level (no chains of chains).
Child vars override parent vars on key collision.

Use --clear to remove inheritance from a profile.`,
	Args: cobra.MinimumNArgs(1),
	RunE: runSetInherit,
}

func init() {
	setInheritCmd.Flags().Bool("clear", false, "Remove inheritance from a profile")
	rootCmd.AddCommand(setTTLCmd)
	rootCmd.AddCommand(setInheritCmd)
}

func runSetTTL(cmd *cobra.Command, args []string) error {
	profileName := args[0]
	durationStr := args[1]

	store, err := getStore()
	if err != nil {
		return err
	}

	p, err := store.Get(profileName)
	if err != nil {
		return fmt.Errorf("profile %q: %w", profileName, err)
	}

	d, err := profile.ParseDuration(durationStr)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", durationStr, err)
	}

	p.TTL = profile.Duration(d)
	if err := store.Put(p); err != nil {
		return err
	}

	if d == 0 {
		fmt.Fprintf(os.Stderr, "TTL disabled for profile %q\n", profileName)
	} else {
		fmt.Fprintf(os.Stderr, "TTL set to %s for profile %q\n", d, profileName)
	}

	return nil
}

func runSetInherit(cmd *cobra.Command, args []string) error {
	profileName := args[0]

	clearFlag, _ := cmd.Flags().GetBool("clear")

	store, err := getStore()
	if err != nil {
		return err
	}

	p, err := store.Get(profileName)
	if err != nil {
		return fmt.Errorf("profile %q: %w", profileName, err)
	}

	if clearFlag {
		p.Inherits = ""
		if err := store.Put(p); err != nil {
			return err
		}
		fmt.Fprintf(os.Stderr, "Inheritance cleared for profile %q\n", profileName)
		return nil
	}

	if len(args) < 2 {
		return fmt.Errorf("usage: envault set-inherit <profile> <parent-profile>\n" +
			"       envault set-inherit <profile> --clear")
	}

	parentName := args[1]

	// Verify parent exists
	_, err = store.Get(parentName)
	if err != nil {
		return fmt.Errorf("parent profile %q: %w", parentName, err)
	}

	// Prevent self-inheritance
	if profileName == parentName {
		return fmt.Errorf("a profile cannot inherit from itself")
	}

	p.Inherits = parentName
	if err := store.Put(p); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Profile %q now inherits from %q\n", profileName, parentName)
	return nil
}

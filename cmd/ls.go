package cmd

import (
	"fmt"
	"sort"
	"time"

	"github.com/Wa-Constellation/envault/internal/profile"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls [profile]",
	Short: "List profiles or keys in a profile",
	Long: `Without arguments: list all profile names with metadata.
With a profile name: list all key names in that profile (NOT values).`,
	Args: cobra.MaximumNArgs(1),
	RunE: runLs,
}

func init() {
	rootCmd.AddCommand(lsCmd)
}

func runLs(cmd *cobra.Command, args []string) error {
	store, err := getStore()
	if err != nil {
		return err
	}

	if len(args) == 0 {
		return listProfiles(store)
	}

	return listKeys(store, args[0])
}

func listProfiles(store *profile.Store) error {
	names, err := store.List()
	if err != nil {
		return err
	}

	if len(names) == 0 {
		fmt.Println("No profiles found.")
		return nil
	}

	sort.Strings(names)

	for _, name := range names {
		p, err := store.Get(name)
		if err != nil {
			fmt.Printf("%-20s  (error: %v)\n", name, err)
			continue
		}

		meta := fmt.Sprintf("%d keys", len(p.Vars))

		if p.Inherits != "" {
			meta += fmt.Sprintf("   inherits:%s", p.Inherits)
		}

		if p.TTL != 0 {
			ttl := time.Duration(p.TTL)
			remaining := ttl - time.Since(p.UpdatedAt)
			if remaining > 0 {
				meta += fmt.Sprintf("   ttl:%s (expires in %s)", ttl, remaining.Round(time.Minute))
			} else {
				meta += fmt.Sprintf("   ttl:%s (EXPIRED)", ttl)
			}
		} else {
			meta += "   ttl:none"
		}

		fmt.Printf("%-20s  %s\n", name, meta)
	}

	return nil
}

func listKeys(store *profile.Store, profileName string) error {
	p, err := store.Get(profileName)
	if err != nil {
		return fmt.Errorf("profile %q: %w", profileName, err)
	}

	fmt.Printf("Profile: %s\n", p.Name)

	if p.Inherits != "" {
		fmt.Printf("Inherits: %s\n", p.Inherits)
	}

	if p.TTL != 0 {
		ttl := time.Duration(p.TTL)
		remaining := ttl - time.Since(p.UpdatedAt)
		if remaining > 0 {
			fmt.Printf("TTL: %s (expires in %s)\n", ttl, remaining.Round(time.Minute))
		} else {
			fmt.Printf("TTL: %s (EXPIRED)\n", ttl)
		}
	}

	fmt.Printf("Created: %s\n", p.CreatedAt.Format(time.RFC3339))
	fmt.Printf("Updated: %s\n", p.UpdatedAt.Format(time.RFC3339))

	// Own keys
	ownKeys := make([]string, 0, len(p.Vars))
	for k := range p.Vars {
		ownKeys = append(ownKeys, k)
	}
	sort.Strings(ownKeys)

	fmt.Printf("Keys (own):\n")
	if len(ownKeys) == 0 {
		fmt.Println("  (none)")
	}
	for _, k := range ownKeys {
		fmt.Printf("  %s\n", k)
	}

	// Inherited keys
	if p.Inherits != "" {
		parent, err := store.Get(p.Inherits)
		if err != nil {
			fmt.Printf("Keys (inherited from %s): (error: %v)\n", p.Inherits, err)
		} else {
			inheritedKeys := make([]string, 0, len(parent.Vars))
			for k := range parent.Vars {
				// Only show inherited keys that are NOT overridden by the child
				if _, overridden := p.Vars[k]; !overridden {
					inheritedKeys = append(inheritedKeys, k)
				}
			}
			sort.Strings(inheritedKeys)

			fmt.Printf("Keys (inherited from %s):\n", p.Inherits)
			if len(inheritedKeys) == 0 {
				fmt.Println("  (none - all overridden)")
			}
			for _, k := range inheritedKeys {
				fmt.Printf("  %s\n", k)
			}
		}
	}

	return nil
}

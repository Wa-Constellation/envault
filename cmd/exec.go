package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/envault/envault/internal/awssts"
	"github.com/envault/envault/internal/profile"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <profile> -- <command> [args...]",
	Short: "Run a command with profile vars injected",
	Long: `Spawn a subprocess with the profile's environment variables injected.
The child inherits the parent's env + profile vars (profile wins on conflict).
Signals (SIGINT, SIGTERM, SIGHUP) are forwarded to the child.
Exit code of envault = exit code of child.`,
	Args:               cobra.MinimumNArgs(1),
	RunE:               runExec,
	DisableFlagParsing: false,
}

func init() {
	execCmd.Flags().String("ttl", "", "Override TTL for this session")
	execCmd.Flags().Bool("no-sts", false, "Skip automatic STS temporary credential generation for AWS keys")
	rootCmd.AddCommand(execCmd)
}

func runExec(cmd *cobra.Command, args []string) error {
	profileName := args[0]

	// Find the command after "--"
	dashIdx := cmd.ArgsLenAtDash()
	if dashIdx == -1 {
		return fmt.Errorf("usage: envault exec <profile> -- <command> [args...]\n" +
			"The -- separator is required before the command.")
	}

	childArgs := args[dashIdx:]
	if len(childArgs) == 0 {
		return fmt.Errorf("no command specified after --")
	}

	store, err := getStore()
	if err != nil {
		return err
	}

	p, err := store.Get(profileName)
	if err != nil {
		return fmt.Errorf("profile %q: %w", profileName, err)
	}

	// Check TTL override
	ttlStr, _ := cmd.Flags().GetString("ttl")
	if ttlStr != "" {
		d, err := profile.ParseDuration(ttlStr)
		if err != nil {
			return fmt.Errorf("invalid TTL %q: %w", ttlStr, err)
		}
		p.TTL = profile.Duration(d)
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
		verboseLog("assuming role %s via STS", vars[awssts.KeyRoleARN])
		if err := awssts.AssumeRole(vars); err != nil {
			return fmt.Errorf("AWS AssumeRole failed: %w\n"+
				"Use --no-sts to skip", err)
		}
		verboseLog("injecting temporary AWS credentials (session token included)")
	}

	verboseLog("injecting %d vars into %s", len(vars), childArgs[0])

	// Build environment: parent env + profile vars (profile wins on conflict).
	// Filter out parent keys that the profile overrides to avoid duplicates.
	env := make([]string, 0, len(os.Environ())+len(vars))
	for _, e := range os.Environ() {
		k := e[:strings.Index(e, "=")+1]
		if k == "" {
			continue
		}
		key := k[:len(k)-1]
		if _, overridden := vars[key]; !overridden {
			env = append(env, e)
		}
	}
	for k, v := range vars {
		env = append(env, k+"="+v)
	}

	// Spawn child
	child := exec.Command(childArgs[0], childArgs[1:]...)
	child.Env = env
	child.Stdin = os.Stdin
	child.Stdout = os.Stdout
	child.Stderr = os.Stderr

	// Forward signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)

	if err := child.Start(); err != nil {
		return fmt.Errorf("failed to start %q: %w", childArgs[0], err)
	}

	go func() {
		for sig := range sigCh {
			_ = child.Process.Signal(sig)
		}
	}()

	err = child.Wait()
	signal.Stop(sigCh)
	close(sigCh)

	// Propagate exit code
	if exitErr, ok := err.(*exec.ExitError); ok {
		os.Exit(exitErr.ExitCode())
	}

	return err
}

package main

import (
	"errors"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/nycjv321/dnsctl/internal/config"
	"github.com/nycjv321/dnsctl/internal/dns"
	"github.com/nycjv321/dnsctl/internal/service"
	"github.com/nycjv321/dnsctl/internal/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dnsctl",
	Short: "Switch DNS profiles and manage local /etc/hosts entries on macOS",
	Long: `dnsctl is a macOS tool for DNS configuration.

Run without arguments to open the interactive TUI for switching DNS profiles.
Use the "hosts" subcommand to manage local /etc/hosts entries from scripts or
agents.`,
	// Default action (no subcommand) launches the TUI, preserving prior behavior.
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command and exits non-zero on error.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}

// runTUI loads config, creates the DNS client, and runs the Bubble Tea program.
func runTUI() error {
	cfg, err := config.Load("")
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	dnsClient, err := dns.NewClient()
	if err != nil {
		if errors.Is(err, dns.ErrNoDNSBackend) {
			fmt.Fprintln(os.Stderr, "Error: no supported DNS management system detected")
			fmt.Fprintln(os.Stderr, "")
			fmt.Fprintln(os.Stderr, "Supported systems:")
			fmt.Fprintln(os.Stderr, "  - macOS with networksetup")
			fmt.Fprintln(os.Stderr, "  - Linux with systemd-resolved (resolvectl)")
			fmt.Fprintln(os.Stderr, "  - Linux with NetworkManager (nmcli)")
			os.Exit(1)
		}
		return fmt.Errorf("creating DNS client: %w", err)
	}

	// Share one DNS client for reads and (in-process) privileged writes.
	resolver := service.NewResolverService(dnsClient, service.NewDirectRunnerWithClient(dnsClient))
	model := tui.NewModel(cfg, resolver)
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("running program: %w", err)
	}
	return nil
}

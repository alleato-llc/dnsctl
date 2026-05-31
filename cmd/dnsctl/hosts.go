package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/nycjv321/dnsctl/internal/hosts"
	"github.com/nycjv321/dnsctl/internal/service"
	"github.com/spf13/cobra"
)

// hosts subcommand flags.
var (
	hostsFile    string
	hostsJSON    bool
	hostsDryRun  bool
	hostsFlush   bool
	hostsAliases []string
	hostsComment string
)

var hostsCmd = &cobra.Command{
	Use:   "hosts",
	Short: "Manage dnsctl-owned entries in /etc/hosts",
	Long: `Manage local hostname mappings in /etc/hosts.

dnsctl only ever edits its own managed block, delimited by sentinel comments.
Lines outside that block (localhost, broadcasthost, and anything you added by
hand) are preserved exactly.

Writing /etc/hosts requires root; run write subcommands with sudo.`,
}

var hostsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List managed host entries",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		entries, err := hostsService().List()
		if err != nil {
			return err
		}
		return output(entries)
	},
}

var hostsGetCmd = &cobra.Command{
	Use:   "get <hostname>",
	Short: "Show the managed entry for a hostname",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		e, ok, err := hostsService().Get(args[0])
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf("no managed entry for %q", args[0])
		}
		return output(e)
	},
}

var hostsAddCmd = &cobra.Command{
	Use:   "add <hostname> <ip>",
	Short: "Add a new host entry (fails if it already exists)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		entry := hosts.Entry{IP: args[1], Hostname: args[0], Aliases: hostsAliases, Comment: hostsComment}
		content, err := hostsService().Add(entry, applyOptions())
		if err != nil {
			if errors.Is(err, service.ErrExists) {
				return fmt.Errorf("%v (use `set` to update)", err)
			}
			return err
		}
		return printIfDryRun(content)
	},
}

var hostsSetCmd = &cobra.Command{
	Use:   "set <hostname> <ip>",
	Short: "Add or update a host entry (idempotent)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		entry := hosts.Entry{IP: args[1], Hostname: args[0], Aliases: hostsAliases, Comment: hostsComment}
		content, err := hostsService().Set(entry, applyOptions())
		if err != nil {
			return err
		}
		return printIfDryRun(content)
	},
}

var hostsRemoveCmd = &cobra.Command{
	Use:     "rm <hostname>",
	Aliases: []string{"remove", "delete"},
	Short:   "Remove a managed host entry",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		content, err := hostsService().Remove(args[0], applyOptions())
		if err != nil {
			if errors.Is(err, service.ErrNotFound) {
				return fmt.Errorf("no managed entry for %q", args[0])
			}
			return err
		}
		return printIfDryRun(content)
	},
}

func init() {
	hostsCmd.PersistentFlags().StringVar(&hostsFile, "file", hosts.DefaultPath, "hosts file to operate on")
	hostsCmd.PersistentFlags().BoolVar(&hostsJSON, "json", false, "output as JSON")

	for _, c := range []*cobra.Command{hostsAddCmd, hostsSetCmd, hostsRemoveCmd} {
		c.Flags().BoolVar(&hostsDryRun, "dry-run", false, "print the resulting file without writing")
		c.Flags().BoolVar(&hostsFlush, "flush", false, "flush the DNS cache after writing")
	}
	for _, c := range []*cobra.Command{hostsAddCmd, hostsSetCmd} {
		c.Flags().StringSliceVar(&hostsAliases, "alias", nil, "additional hostname for the same IP (repeatable)")
		c.Flags().StringVar(&hostsComment, "comment", "", "trailing comment for the entry")
	}

	hostsCmd.AddCommand(hostsListCmd, hostsGetCmd, hostsAddCmd, hostsSetCmd, hostsRemoveCmd)
	rootCmd.AddCommand(hostsCmd)
}

// hostsService builds the shared facade, routing privileged writes in-process
// (root) or through the helper (non-root) via chooseRunner.
func hostsService() *service.HostsService {
	return service.NewHostsService(hostsFile, chooseRunner())
}

// applyOptions maps the shared write flags onto the service options.
func applyOptions() service.ApplyOptions {
	return service.ApplyOptions{DryRun: hostsDryRun, Flush: hostsFlush}
}

// printIfDryRun writes the previewed file content to stdout under --dry-run;
// successful writes produce no output.
func printIfDryRun(content []byte) error {
	if hostsDryRun {
		_, err := os.Stdout.Write(content)
		return err
	}
	return nil
}

// output prints entries as JSON or a human-readable table depending on --json.
// v is either a hosts.Entry or a []hosts.Entry.
func output(v any) error {
	if hostsJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(v)
	}

	var entries []hosts.Entry
	switch t := v.(type) {
	case hosts.Entry:
		entries = []hosts.Entry{t}
	case []hosts.Entry:
		entries = t
	}

	if len(entries) == 0 {
		fmt.Println("No managed entries.")
		return nil
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "IP\tHOSTNAME\tALIASES\tCOMMENT")
	for _, e := range entries {
		aliases := ""
		for i, a := range e.Aliases {
			if i > 0 {
				aliases += ","
			}
			aliases += a
		}
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", e.IP, e.Hostname, aliases, e.Comment)
	}
	return w.Flush()
}

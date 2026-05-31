package main

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/nycjv321/dnsctl/internal/dns"
	"github.com/nycjv321/dnsctl/internal/hosts"
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
		doc, err := hosts.NewStore(hostsFile).Load()
		if err != nil {
			return err
		}
		return output(doc.List())
	},
}

var hostsGetCmd = &cobra.Command{
	Use:   "get <hostname>",
	Short: "Show the managed entry for a hostname",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		doc, err := hosts.NewStore(hostsFile).Load()
		if err != nil {
			return err
		}
		e, ok := doc.Get(args[0])
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
		return upsert(args[0], args[1], false)
	},
}

var hostsSetCmd = &cobra.Command{
	Use:   "set <hostname> <ip>",
	Short: "Add or update a host entry (idempotent)",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return upsert(args[0], args[1], true)
	},
}

var hostsRemoveCmd = &cobra.Command{
	Use:     "rm <hostname>",
	Aliases: []string{"remove", "delete"},
	Short:   "Remove a managed host entry",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		store := hosts.NewStore(hostsFile)
		doc, err := store.Load()
		if err != nil {
			return err
		}
		if !doc.Remove(args[0]) {
			return fmt.Errorf("no managed entry for %q", args[0])
		}
		return commit(store, doc)
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

// upsert adds or sets an entry. When allowExisting is false (add), an existing
// hostname is an error.
func upsert(hostname, ip string, allowExisting bool) error {
	store := hosts.NewStore(hostsFile)
	doc, err := store.Load()
	if err != nil {
		return err
	}

	if !allowExisting {
		if _, exists := doc.Get(hostname); exists {
			return fmt.Errorf("entry for %q already exists (use `set` to update)", hostname)
		}
	}

	entry := hosts.Entry{IP: ip, Hostname: hostname, Aliases: hostsAliases, Comment: hostsComment}
	if err := entry.Validate(); err != nil {
		return err
	}
	doc.Set(entry)
	return commit(store, doc)
}

// commit writes the document (or prints it under --dry-run) and optionally
// flushes the DNS cache.
func commit(store *hosts.Store, doc *hosts.Document) error {
	if hostsDryRun {
		os.Stdout.Write(doc.Render())
		return nil
	}
	if err := store.Save(doc); err != nil {
		return err
	}
	if hostsFlush {
		client, err := dns.NewClient()
		if err != nil {
			return fmt.Errorf("flush: %w", err)
		}
		if err := client.FlushCache(); err != nil {
			return fmt.Errorf("flush: %w", err)
		}
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

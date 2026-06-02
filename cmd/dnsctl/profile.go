package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/nycjv321/dnsctl/internal/dns"
	"github.com/nycjv321/dnsctl/internal/service"
	"github.com/spf13/cobra"
)

// profile subcommand flags.
var (
	profileConfig   string // config file override ("" = default location)
	profileJSON     bool
	profileService  string   // target network service for `apply`
	profileServers  []string // servers for `add`/`set`
	profileDescript string
	profileDHCP     bool
)

var profileCmd = &cobra.Command{
	Use:   "profile",
	Short: "List, apply, and edit DNS profiles",
	Long: `Manage named DNS profiles.

Profiles live in the user config (~/.config/dnsctl/config.yaml) and map a name
to a set of DNS servers (or DHCP). "apply" switches a network service to a
profile; applying requires root (run with sudo) or the dnsctl-helper. Listing
and editing the profile definitions only touch the user-owned config file and
need no privileges.`,
}

var profileListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured profiles",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		profiles, err := profileSvc().List()
		if err != nil {
			return err
		}
		return outputProfiles(profiles)
	},
}

var profileApplyCmd = &cobra.Command{
	Use:   "apply <name>",
	Short: "Apply a profile to a network service",
	Long: `Apply a profile, switching a network service's DNS servers.

With no --service, the target is the config's default_service, then the active
(default-route) service. Requires root or the dnsctl-helper.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := profileSvc().Apply(args[0], profileService); err != nil {
			if errors.Is(err, service.ErrNoProfile) {
				return fmt.Errorf("%v (run `dnsctl profile list`)", err)
			}
			return err
		}
		if !profileJSON {
			fmt.Printf("Applied profile %q.\n", args[0])
		}
		return nil
	},
}

var profileAddCmd = &cobra.Command{
	Use:   "add <name>",
	Short: "Create or update a profile definition",
	Long: `Create or update a profile.

Provide --server (repeatable) for a specific resolver set, or --dhcp for a
profile that reverts the service to DHCP-provided DNS. This edits the user
config only; it does not change any active resolver settings.`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if !profileDHCP && len(profileServers) == 0 {
			return errors.New("provide --server (repeatable) or --dhcp")
		}
		p := service.Profile{
			Name:        args[0],
			Description: profileDescript,
			Servers:     profileServers,
			DHCP:        profileDHCP,
		}
		if err := profileSvc().Save(p); err != nil {
			return err
		}
		if profileJSON {
			return outputProfiles([]service.Profile{p})
		}
		fmt.Printf("Saved profile %q.\n", args[0])
		return nil
	},
}

var profileRemoveCmd = &cobra.Command{
	Use:     "rm <name>",
	Aliases: []string{"remove", "delete"},
	Short:   "Delete a profile definition",
	Args:    cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := profileSvc().Delete(args[0]); err != nil {
			if errors.Is(err, service.ErrNoProfile) {
				return fmt.Errorf("no such profile %q", args[0])
			}
			return err
		}
		if !profileJSON {
			fmt.Printf("Deleted profile %q.\n", args[0])
		}
		return nil
	},
}

func init() {
	profileCmd.PersistentFlags().StringVar(&profileConfig, "config", "", "config file to operate on (default ~/.config/dnsctl/config.yaml)")
	profileCmd.PersistentFlags().BoolVar(&profileJSON, "json", false, "output as JSON")

	profileApplyCmd.Flags().StringVar(&profileService, "service", "", "network service to apply to (default: config default, then active service)")

	profileAddCmd.Flags().StringSliceVar(&profileServers, "server", nil, "DNS server IP (repeatable)")
	profileAddCmd.Flags().StringVar(&profileDescript, "description", "", "human-readable description")
	profileAddCmd.Flags().BoolVar(&profileDHCP, "dhcp", false, "profile reverts the service to DHCP-provided DNS")

	profileCmd.AddCommand(profileListCmd, profileApplyCmd, profileAddCmd, profileRemoveCmd)
	rootCmd.AddCommand(profileCmd)
}

// profileSvc builds the profile facade. Reads/edits hit the config file; Apply
// routes privileged resolver writes via chooseRunner (in-process when root,
// else through the helper). When no DNS backend is available, Apply errors but
// list/edit still work.
func profileSvc() *service.ProfileService {
	var resolver *service.ResolverService
	if client, err := dns.NewClient(); err == nil {
		resolver = service.NewResolverService(client, chooseRunner())
	}
	return service.NewProfileService(profileConfig, resolver)
}

// outputProfiles prints profiles as JSON or a human-readable table.
func outputProfiles(profiles []service.Profile) error {
	if profileJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(profiles)
	}
	if len(profiles) == 0 {
		fmt.Println("No profiles configured.")
		return nil
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "NAME\tSERVERS\tDESCRIPTION")
	for _, p := range profiles {
		servers := strings.Join(p.Servers, ",")
		if p.IsDHCP() {
			servers = "DHCP"
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", p.Name, servers, p.Description)
	}
	return w.Flush()
}

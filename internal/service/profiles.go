package service

import (
	"errors"
	"fmt"

	"github.com/nycjv321/dnsctl/internal/config"
)

// ErrNoProfile is returned when a named profile does not exist in the config.
var ErrNoProfile = errors.New("no such profile")

// Profile is a named DNS profile as surfaced to the CLI/GUI. It mirrors
// config.Profile but carries the Name (the config map key) and uses JSON tags
// so it doubles as the Wails (Go -> TypeScript) binding type.
type Profile struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Servers     []string `json:"servers"`
	DHCP        bool     `json:"dhcp"`
}

// IsDHCP reports whether applying this profile reverts to DHCP-provided DNS.
func (p Profile) IsDHCP() bool {
	return p.DHCP || len(p.Servers) == 0
}

// ProfileService manages named DNS profiles stored in the user's config file
// and applies them to network services via the resolver. It is the shared
// facade behind both `dnsctl profile` and the GUI's Profiles view.
//
// Profile definitions live in ~/.config/dnsctl/config.yaml, which is
// user-owned: List/Save/Delete read and write it unprivileged. Only Apply has
// a root-only side effect (changing resolver config), which it delegates to the
// ResolverService (and thus the PrivilegedRunner).
type ProfileService struct {
	path     string // config file path; "" means the default location
	resolver *ResolverService
}

// NewProfileService returns a ProfileService backed by the config file at path
// (empty means config.DefaultConfigPath). resolver performs profile
// application; it may be nil, in which case Apply reports an error but the
// read/edit operations still work.
func NewProfileService(path string, resolver *ResolverService) *ProfileService {
	return &ProfileService{path: path, resolver: resolver}
}

// List returns the configured profiles, sorted by name. The slice is non-nil so
// it marshals to a JSON array rather than null.
func (s *ProfileService) List() ([]Profile, error) {
	cfg, err := config.Load(s.path)
	if err != nil {
		return nil, err
	}
	out := make([]Profile, 0, len(cfg.Profiles))
	for _, name := range cfg.ProfileNames() {
		out = append(out, toProfile(name, cfg.Profiles[name]))
	}
	return out, nil
}

// Get returns a single profile by name.
func (s *ProfileService) Get(name string) (Profile, error) {
	cfg, err := config.Load(s.path)
	if err != nil {
		return Profile{}, err
	}
	p, ok := cfg.GetProfile(name)
	if !ok {
		return Profile{}, fmt.Errorf("%w: %q", ErrNoProfile, name)
	}
	return toProfile(name, p), nil
}

// Apply applies the named profile to a network service. When service is empty
// it falls back to the config's default_service, and then to the active
// (default-route) service. DHCP profiles clear DNS; others set the servers.
// The DNS cache is flushed afterward when the config's flush_cache is set
// (best-effort, mirroring the TUI).
func (s *ProfileService) Apply(name, service string) error {
	if s.resolver == nil {
		return errors.New("no DNS backend available")
	}
	cfg, err := config.Load(s.path)
	if err != nil {
		return err
	}
	p, ok := cfg.GetProfile(name)
	if !ok {
		return fmt.Errorf("%w: %q", ErrNoProfile, name)
	}

	service = s.resolveService(cfg, service)
	if service == "" {
		return errors.New("no network service specified and no default could be determined")
	}

	if p.IsDHCP() {
		err = s.resolver.Clear(service)
	} else {
		err = s.resolver.Set(service, p.Servers)
	}
	if err != nil {
		return fmt.Errorf("applying profile %q to %q: %w", name, service, err)
	}

	if cfg.Settings.FlushCache {
		_ = s.resolver.Flush() // best-effort, mirroring the TUI
	}
	return nil
}

// resolveService picks the target service: an explicit choice wins, then the
// config default, then the active (default-route) service.
func (s *ProfileService) resolveService(cfg *config.Config, service string) string {
	if service != "" {
		return service
	}
	if cfg.DefaultService != "" {
		return cfg.DefaultService
	}
	if primary, err := s.resolver.PrimaryService(); err == nil {
		return primary
	}
	return ""
}

// Save upserts a profile definition (create or edit) and persists the config.
// This writes the user-owned config file and needs no privileges.
func (s *ProfileService) Save(p Profile) error {
	if p.Name == "" {
		return errors.New("profile name is required")
	}
	cfg, err := config.Load(s.path)
	if err != nil {
		return err
	}
	if cfg.Profiles == nil {
		cfg.Profiles = make(map[string]config.Profile)
	}
	cfg.Profiles[p.Name] = config.Profile{
		Description: p.Description,
		Servers:     p.Servers,
		DHCP:        p.DHCP,
	}
	return cfg.Save(s.path)
}

// Delete removes a profile definition and persists the config.
func (s *ProfileService) Delete(name string) error {
	cfg, err := config.Load(s.path)
	if err != nil {
		return err
	}
	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("%w: %q", ErrNoProfile, name)
	}
	delete(cfg.Profiles, name)
	return cfg.Save(s.path)
}

func toProfile(name string, p config.Profile) Profile {
	// Non-nil servers so it marshals to [] rather than null (matches the hosts
	// list convention and keeps the TS binding's array type honest).
	servers := p.Servers
	if servers == nil {
		servers = []string{}
	}
	return Profile{
		Name:        name,
		Description: p.Description,
		Servers:     servers,
		DHCP:        p.IsDHCP(),
	}
}

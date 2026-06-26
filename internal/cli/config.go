package cli

import "github.com/rnwolfe/ufi/internal/errs"

// Declarative config surface (spec: Config command surface). Reads emit placeholder
// envelopes; writes are high-stakes, so they DON'T execute directly — `previewConfig`
// emits a plan + hash and the operator runs `ufi apply <hash>` (reviewed-artifact, §2).
// cli-implement wires the real CRUD calls + plan persistence/execution.
//
// --data accepts a path, "-" for stdin, or inline JSON; cli-implement parses + validates it.

// --- network ----------------------------------------------------------------

type NetworkCmd struct {
	List   NetworkListCmd   `cmd:"" help:"List networks (VLAN/LAN)."`
	Get    NetworkGetCmd    `cmd:"" help:"Get one network by id."`
	Create NetworkWriteCmd  `cmd:"" help:"Create a network (config mutation)."`
	Update NetworkUpdateCmd `cmd:"" help:"Update a network (config mutation)."`
	Delete NetworkDeleteCmd `cmd:"" help:"Delete a network (config mutation)."`
}

type NetworkListCmd struct{}

func (c *NetworkListCmd) Run(rt *Runtime) error { return rt.emitEmptyList() }

type NetworkGetCmd struct {
	ID string `arg:"" help:"Network id."`
}

func (c *NetworkGetCmd) Run(rt *Runtime) error { return rt.emitPlaceholderObject() }

type NetworkWriteCmd struct {
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *NetworkWriteCmd) Run(rt *Runtime) error {
	return rt.previewConfig("network create", map[string]any{"data": c.Data})
}

type NetworkUpdateCmd struct {
	ID   string `arg:"" help:"Network id."`
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *NetworkUpdateCmd) Run(rt *Runtime) error {
	return rt.previewConfig("network update", map[string]any{"id": c.ID, "data": c.Data})
}

type NetworkDeleteCmd struct {
	ID string `arg:"" help:"Network id."`
}

func (c *NetworkDeleteCmd) Run(rt *Runtime) error {
	return rt.previewConfig("network delete", map[string]any{"id": c.ID})
}

// --- firewall (Zone-Based Firewall) -----------------------------------------
// Reads require ZBF enabled; cli-implement maps the upstream
// 400 api.firewall.zone-based-firewall-not-configured to UNSUPPORTED with remediation.

type FirewallCmd struct {
	Policy FirewallPolicyCmd `cmd:"" help:"Manage firewall policies (rules)."`
	Zone   FirewallZoneCmd   `cmd:"" help:"Manage firewall zones."`
}

type FirewallPolicyCmd struct {
	List    FirewallPolicyListCmd    `cmd:"" help:"List firewall policies."`
	Get     FirewallPolicyGetCmd     `cmd:"" help:"Get one firewall policy."`
	Create  FirewallPolicyWriteCmd   `cmd:"" help:"Create a firewall policy (config mutation)."`
	Update  FirewallPolicyUpdateCmd  `cmd:"" help:"Update a firewall policy (config mutation)."`
	Delete  FirewallPolicyDeleteCmd  `cmd:"" help:"Delete a firewall policy (config mutation)."`
	Reorder FirewallPolicyReorderCmd `cmd:"" help:"Reorder firewall policies (config mutation)."`
}

type FirewallPolicyListCmd struct{}

func (c *FirewallPolicyListCmd) Run(rt *Runtime) error { return rt.emitEmptyList() }

type FirewallPolicyGetCmd struct {
	ID string `arg:"" help:"Firewall policy id."`
}

func (c *FirewallPolicyGetCmd) Run(rt *Runtime) error { return rt.emitPlaceholderObject() }

type FirewallPolicyWriteCmd struct {
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *FirewallPolicyWriteCmd) Run(rt *Runtime) error {
	return rt.previewConfig("firewall policy create", map[string]any{"data": c.Data})
}

type FirewallPolicyUpdateCmd struct {
	ID   string `arg:"" help:"Firewall policy id."`
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *FirewallPolicyUpdateCmd) Run(rt *Runtime) error {
	return rt.previewConfig("firewall policy update", map[string]any{"id": c.ID, "data": c.Data})
}

type FirewallPolicyDeleteCmd struct {
	ID string `arg:"" help:"Firewall policy id."`
}

func (c *FirewallPolicyDeleteCmd) Run(rt *Runtime) error {
	return rt.previewConfig("firewall policy delete", map[string]any{"id": c.ID})
}

type FirewallPolicyReorderCmd struct {
	IDs []string `arg:"" help:"Policy ids in the desired order."`
}

func (c *FirewallPolicyReorderCmd) Run(rt *Runtime) error {
	return rt.previewConfig("firewall policy reorder", map[string]any{"order": c.IDs})
}

type FirewallZoneCmd struct {
	List   FirewallZoneListCmd   `cmd:"" help:"List firewall zones."`
	Get    FirewallZoneGetCmd    `cmd:"" help:"Get one firewall zone."`
	Create FirewallZoneWriteCmd  `cmd:"" help:"Create a firewall zone (config mutation)."`
	Update FirewallZoneUpdateCmd `cmd:"" help:"Update a firewall zone (config mutation)."`
	Delete FirewallZoneDeleteCmd `cmd:"" help:"Delete a firewall zone (config mutation)."`
}

type FirewallZoneListCmd struct{}

func (c *FirewallZoneListCmd) Run(rt *Runtime) error { return rt.emitEmptyList() }

type FirewallZoneGetCmd struct {
	ID string `arg:"" help:"Firewall zone id."`
}

func (c *FirewallZoneGetCmd) Run(rt *Runtime) error { return rt.emitPlaceholderObject() }

type FirewallZoneWriteCmd struct {
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *FirewallZoneWriteCmd) Run(rt *Runtime) error {
	return rt.previewConfig("firewall zone create", map[string]any{"data": c.Data})
}

type FirewallZoneUpdateCmd struct {
	ID   string `arg:"" help:"Firewall zone id."`
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *FirewallZoneUpdateCmd) Run(rt *Runtime) error {
	return rt.previewConfig("firewall zone update", map[string]any{"id": c.ID, "data": c.Data})
}

type FirewallZoneDeleteCmd struct {
	ID string `arg:"" help:"Firewall zone id."`
}

func (c *FirewallZoneDeleteCmd) Run(rt *Runtime) error {
	return rt.previewConfig("firewall zone delete", map[string]any{"id": c.ID})
}

// --- acl --------------------------------------------------------------------

type AclCmd struct {
	List    AclListCmd    `cmd:"" help:"List ACL rules."`
	Get     AclGetCmd     `cmd:"" help:"Get one ACL rule."`
	Create  AclWriteCmd   `cmd:"" help:"Create an ACL rule (config mutation)."`
	Update  AclUpdateCmd  `cmd:"" help:"Update an ACL rule (config mutation)."`
	Delete  AclDeleteCmd  `cmd:"" help:"Delete an ACL rule (config mutation)."`
	Reorder AclReorderCmd `cmd:"" help:"Reorder ACL rules (config mutation)."`
}

type AclListCmd struct{}

func (c *AclListCmd) Run(rt *Runtime) error { return rt.emitEmptyList() }

type AclGetCmd struct {
	ID string `arg:"" help:"ACL rule id."`
}

func (c *AclGetCmd) Run(rt *Runtime) error { return rt.emitPlaceholderObject() }

type AclWriteCmd struct {
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *AclWriteCmd) Run(rt *Runtime) error {
	return rt.previewConfig("acl create", map[string]any{"data": c.Data})
}

type AclUpdateCmd struct {
	ID   string `arg:"" help:"ACL rule id."`
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *AclUpdateCmd) Run(rt *Runtime) error {
	return rt.previewConfig("acl update", map[string]any{"id": c.ID, "data": c.Data})
}

type AclDeleteCmd struct {
	ID string `arg:"" help:"ACL rule id."`
}

func (c *AclDeleteCmd) Run(rt *Runtime) error {
	return rt.previewConfig("acl delete", map[string]any{"id": c.ID})
}

type AclReorderCmd struct {
	IDs []string `arg:"" help:"ACL rule ids in the desired order."`
}

func (c *AclReorderCmd) Run(rt *Runtime) error {
	return rt.previewConfig("acl reorder", map[string]any{"order": c.IDs})
}

// --- dns --------------------------------------------------------------------

type DnsCmd struct {
	Policy DnsPolicyCmd `cmd:"" help:"Manage DNS policies."`
}

type DnsPolicyCmd struct {
	List   DnsPolicyListCmd   `cmd:"" help:"List DNS policies."`
	Get    DnsPolicyGetCmd    `cmd:"" help:"Get one DNS policy."`
	Create DnsPolicyWriteCmd  `cmd:"" help:"Create a DNS policy (config mutation)."`
	Update DnsPolicyUpdateCmd `cmd:"" help:"Update a DNS policy (config mutation)."`
	Delete DnsPolicyDeleteCmd `cmd:"" help:"Delete a DNS policy (config mutation)."`
}

type DnsPolicyListCmd struct{}

func (c *DnsPolicyListCmd) Run(rt *Runtime) error { return rt.emitEmptyList() }

type DnsPolicyGetCmd struct {
	ID string `arg:"" help:"DNS policy id."`
}

func (c *DnsPolicyGetCmd) Run(rt *Runtime) error { return rt.emitPlaceholderObject() }

type DnsPolicyWriteCmd struct {
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *DnsPolicyWriteCmd) Run(rt *Runtime) error {
	return rt.previewConfig("dns policy create", map[string]any{"data": c.Data})
}

type DnsPolicyUpdateCmd struct {
	ID   string `arg:"" help:"DNS policy id."`
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *DnsPolicyUpdateCmd) Run(rt *Runtime) error {
	return rt.previewConfig("dns policy update", map[string]any{"id": c.ID, "data": c.Data})
}

type DnsPolicyDeleteCmd struct {
	ID string `arg:"" help:"DNS policy id."`
}

func (c *DnsPolicyDeleteCmd) Run(rt *Runtime) error {
	return rt.previewConfig("dns policy delete", map[string]any{"id": c.ID})
}

// --- traffic-list -----------------------------------------------------------

type TrafficListCmd struct {
	List   TrafficListListCmd   `cmd:"" help:"List traffic-matching lists."`
	Get    TrafficListGetCmd    `cmd:"" help:"Get one traffic-matching list."`
	Create TrafficListWriteCmd  `cmd:"" help:"Create a traffic-matching list (config mutation)."`
	Update TrafficListUpdateCmd `cmd:"" help:"Update a traffic-matching list (config mutation)."`
	Delete TrafficListDeleteCmd `cmd:"" help:"Delete a traffic-matching list (config mutation)."`
}

type TrafficListListCmd struct{}

func (c *TrafficListListCmd) Run(rt *Runtime) error { return rt.emitEmptyList() }

type TrafficListGetCmd struct {
	ID string `arg:"" help:"Traffic-matching list id."`
}

func (c *TrafficListGetCmd) Run(rt *Runtime) error { return rt.emitPlaceholderObject() }

type TrafficListWriteCmd struct {
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *TrafficListWriteCmd) Run(rt *Runtime) error {
	return rt.previewConfig("traffic-list create", map[string]any{"data": c.Data})
}

type TrafficListUpdateCmd struct {
	ID   string `arg:"" help:"Traffic-matching list id."`
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *TrafficListUpdateCmd) Run(rt *Runtime) error {
	return rt.previewConfig("traffic-list update", map[string]any{"id": c.ID, "data": c.Data})
}

type TrafficListDeleteCmd struct {
	ID string `arg:"" help:"Traffic-matching list id."`
}

func (c *TrafficListDeleteCmd) Run(rt *Runtime) error {
	return rt.previewConfig("traffic-list delete", map[string]any{"id": c.ID})
}

// --- apply ------------------------------------------------------------------

// ApplyCmd executes a previously previewed config plan by hash (reviewed-artifact, §2).
// Scaffold: no plans are persisted yet, so any hash is unknown → USAGE. cli-implement wires
// plan persistence ($XDG_STATE_HOME/ufi/plans/<hash>.json) and execution.
type ApplyCmd struct {
	Hash string `arg:"" help:"Plan hash from a prior config --dry-run preview."`
}

func (c *ApplyCmd) Run(rt *Runtime) error {
	if err := rt.Guard("apply"); err != nil {
		return err
	}
	return errs.New(errs.ExitUsage, "PLAN_NOT_FOUND",
		"no persisted plan for hash "+c.Hash,
		"config plan persistence + execution is wired by cli-implement; re-run the config command with --dry-run to produce a plan")
}

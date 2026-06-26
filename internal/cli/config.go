package cli

import (
	"encoding/json"
	"net/http"

	"github.com/rnwolfe/ufi/internal/errs"
	"github.com/rnwolfe/ufi/internal/plan"
)

// Declarative config surface (spec: Config command surface). Reads use the normal client path;
// writes are high-stakes, so they DON'T execute directly — `configWrite` persists a plan + hash
// and the operator runs `ufi apply <hash>` (reviewed-artifact, contract §2). --data accepts a
// path, "-" for stdin, or inline JSON; it's validated as JSON before a plan is written.

// --- network ----------------------------------------------------------------

type NetworkCmd struct {
	List   NetworkListCmd   `cmd:"" help:"List networks (VLAN/LAN)."`
	Get    NetworkGetCmd    `cmd:"" help:"Get one network by id."`
	Create NetworkCreateCmd `cmd:"" help:"Create a network (config mutation)."`
	Update NetworkUpdateCmd `cmd:"" help:"Update a network (config mutation)."`
	Delete NetworkDeleteCmd `cmd:"" help:"Delete a network (config mutation)."`
}

type NetworkListCmd struct{}

func (c *NetworkListCmd) Run(rt *Runtime) error { return rt.siteList("networks") }

type NetworkGetCmd struct {
	ID string `arg:"" help:"Network id."`
}

func (c *NetworkGetCmd) Run(rt *Runtime) error { return rt.siteGet("networks/" + c.ID) }

type NetworkCreateCmd struct {
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *NetworkCreateCmd) Run(rt *Runtime) error {
	return rt.configCreate("network create", "networks", c.Data)
}

type NetworkUpdateCmd struct {
	ID   string `arg:"" help:"Network id."`
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *NetworkUpdateCmd) Run(rt *Runtime) error {
	return rt.configUpdate("network update", "networks/"+c.ID, c.ID, c.Data)
}

type NetworkDeleteCmd struct {
	ID string `arg:"" help:"Network id."`
}

func (c *NetworkDeleteCmd) Run(rt *Runtime) error {
	return rt.configDelete("network delete", "networks/"+c.ID, c.ID)
}

// --- firewall (Zone-Based Firewall) -----------------------------------------

type FirewallCmd struct {
	Policy FirewallPolicyCmd `cmd:"" help:"Manage firewall policies (rules)."`
	Zone   FirewallZoneCmd   `cmd:"" help:"Manage firewall zones."`
}

type FirewallPolicyCmd struct {
	List    FirewallPolicyListCmd    `cmd:"" help:"List firewall policies."`
	Get     FirewallPolicyGetCmd     `cmd:"" help:"Get one firewall policy."`
	Create  FirewallPolicyCreateCmd  `cmd:"" help:"Create a firewall policy (config mutation)."`
	Update  FirewallPolicyUpdateCmd  `cmd:"" help:"Update a firewall policy (config mutation)."`
	Delete  FirewallPolicyDeleteCmd  `cmd:"" help:"Delete a firewall policy (config mutation)."`
	Reorder FirewallPolicyReorderCmd `cmd:"" help:"Reorder firewall policies (config mutation)."`
}

type FirewallPolicyListCmd struct{}

func (c *FirewallPolicyListCmd) Run(rt *Runtime) error { return rt.siteList("firewall/policies") }

type FirewallPolicyGetCmd struct {
	ID string `arg:"" help:"Firewall policy id."`
}

func (c *FirewallPolicyGetCmd) Run(rt *Runtime) error { return rt.siteGet("firewall/policies/" + c.ID) }

type FirewallPolicyCreateCmd struct {
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *FirewallPolicyCreateCmd) Run(rt *Runtime) error {
	return rt.configCreate("firewall policy create", "firewall/policies", c.Data)
}

type FirewallPolicyUpdateCmd struct {
	ID   string `arg:"" help:"Firewall policy id."`
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *FirewallPolicyUpdateCmd) Run(rt *Runtime) error {
	return rt.configUpdate("firewall policy update", "firewall/policies/"+c.ID, c.ID, c.Data)
}

type FirewallPolicyDeleteCmd struct {
	ID string `arg:"" help:"Firewall policy id."`
}

func (c *FirewallPolicyDeleteCmd) Run(rt *Runtime) error {
	return rt.configDelete("firewall policy delete", "firewall/policies/"+c.ID, c.ID)
}

type FirewallPolicyReorderCmd struct {
	IDs []string `arg:"" name:"id" help:"Policy ids in the desired order."`
}

func (c *FirewallPolicyReorderCmd) Run(rt *Runtime) error {
	return rt.configReorder("firewall policy reorder", "firewall/policies/ordering", c.IDs)
}

type FirewallZoneCmd struct {
	List   FirewallZoneListCmd   `cmd:"" help:"List firewall zones."`
	Get    FirewallZoneGetCmd    `cmd:"" help:"Get one firewall zone."`
	Create FirewallZoneCreateCmd `cmd:"" help:"Create a firewall zone (config mutation)."`
	Update FirewallZoneUpdateCmd `cmd:"" help:"Update a firewall zone (config mutation)."`
	Delete FirewallZoneDeleteCmd `cmd:"" help:"Delete a firewall zone (config mutation)."`
}

type FirewallZoneListCmd struct{}

func (c *FirewallZoneListCmd) Run(rt *Runtime) error { return rt.siteList("firewall/zones") }

type FirewallZoneGetCmd struct {
	ID string `arg:"" help:"Firewall zone id."`
}

func (c *FirewallZoneGetCmd) Run(rt *Runtime) error { return rt.siteGet("firewall/zones/" + c.ID) }

type FirewallZoneCreateCmd struct {
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *FirewallZoneCreateCmd) Run(rt *Runtime) error {
	return rt.configCreate("firewall zone create", "firewall/zones", c.Data)
}

type FirewallZoneUpdateCmd struct {
	ID   string `arg:"" help:"Firewall zone id."`
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *FirewallZoneUpdateCmd) Run(rt *Runtime) error {
	return rt.configUpdate("firewall zone update", "firewall/zones/"+c.ID, c.ID, c.Data)
}

type FirewallZoneDeleteCmd struct {
	ID string `arg:"" help:"Firewall zone id."`
}

func (c *FirewallZoneDeleteCmd) Run(rt *Runtime) error {
	return rt.configDelete("firewall zone delete", "firewall/zones/"+c.ID, c.ID)
}

// --- acl --------------------------------------------------------------------

type AclCmd struct {
	List    AclListCmd    `cmd:"" help:"List ACL rules."`
	Get     AclGetCmd     `cmd:"" help:"Get one ACL rule."`
	Create  AclCreateCmd  `cmd:"" help:"Create an ACL rule (config mutation)."`
	Update  AclUpdateCmd  `cmd:"" help:"Update an ACL rule (config mutation)."`
	Delete  AclDeleteCmd  `cmd:"" help:"Delete an ACL rule (config mutation)."`
	Reorder AclReorderCmd `cmd:"" help:"Reorder ACL rules (config mutation)."`
}

type AclListCmd struct{}

func (c *AclListCmd) Run(rt *Runtime) error { return rt.siteList("acl-rules") }

type AclGetCmd struct {
	ID string `arg:"" help:"ACL rule id."`
}

func (c *AclGetCmd) Run(rt *Runtime) error { return rt.siteGet("acl-rules/" + c.ID) }

type AclCreateCmd struct {
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *AclCreateCmd) Run(rt *Runtime) error {
	return rt.configCreate("acl create", "acl-rules", c.Data)
}

type AclUpdateCmd struct {
	ID   string `arg:"" help:"ACL rule id."`
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *AclUpdateCmd) Run(rt *Runtime) error {
	return rt.configUpdate("acl update", "acl-rules/"+c.ID, c.ID, c.Data)
}

type AclDeleteCmd struct {
	ID string `arg:"" help:"ACL rule id."`
}

func (c *AclDeleteCmd) Run(rt *Runtime) error {
	return rt.configDelete("acl delete", "acl-rules/"+c.ID, c.ID)
}

type AclReorderCmd struct {
	IDs []string `arg:"" name:"id" help:"ACL rule ids in the desired order."`
}

func (c *AclReorderCmd) Run(rt *Runtime) error {
	return rt.configReorder("acl reorder", "acl-rules/ordering", c.IDs)
}

// --- dns --------------------------------------------------------------------

type DnsCmd struct {
	Policy DnsPolicyCmd `cmd:"" help:"Manage DNS policies."`
}

type DnsPolicyCmd struct {
	List   DnsPolicyListCmd   `cmd:"" help:"List DNS policies."`
	Get    DnsPolicyGetCmd    `cmd:"" help:"Get one DNS policy."`
	Create DnsPolicyCreateCmd `cmd:"" help:"Create a DNS policy (config mutation)."`
	Update DnsPolicyUpdateCmd `cmd:"" help:"Update a DNS policy (config mutation)."`
	Delete DnsPolicyDeleteCmd `cmd:"" help:"Delete a DNS policy (config mutation)."`
}

type DnsPolicyListCmd struct{}

func (c *DnsPolicyListCmd) Run(rt *Runtime) error { return rt.siteList("dns/policies") }

type DnsPolicyGetCmd struct {
	ID string `arg:"" help:"DNS policy id."`
}

func (c *DnsPolicyGetCmd) Run(rt *Runtime) error { return rt.siteGet("dns/policies/" + c.ID) }

type DnsPolicyCreateCmd struct {
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *DnsPolicyCreateCmd) Run(rt *Runtime) error {
	return rt.configCreate("dns policy create", "dns/policies", c.Data)
}

type DnsPolicyUpdateCmd struct {
	ID   string `arg:"" help:"DNS policy id."`
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *DnsPolicyUpdateCmd) Run(rt *Runtime) error {
	return rt.configUpdate("dns policy update", "dns/policies/"+c.ID, c.ID, c.Data)
}

type DnsPolicyDeleteCmd struct {
	ID string `arg:"" help:"DNS policy id."`
}

func (c *DnsPolicyDeleteCmd) Run(rt *Runtime) error {
	return rt.configDelete("dns policy delete", "dns/policies/"+c.ID, c.ID)
}

// --- traffic-list -----------------------------------------------------------

type TrafficListCmd struct {
	List   TrafficListListCmd   `cmd:"" help:"List traffic-matching lists."`
	Get    TrafficListGetCmd    `cmd:"" help:"Get one traffic-matching list."`
	Create TrafficListCreateCmd `cmd:"" help:"Create a traffic-matching list (config mutation)."`
	Update TrafficListUpdateCmd `cmd:"" help:"Update a traffic-matching list (config mutation)."`
	Delete TrafficListDeleteCmd `cmd:"" help:"Delete a traffic-matching list (config mutation)."`
}

type TrafficListListCmd struct{}

func (c *TrafficListListCmd) Run(rt *Runtime) error { return rt.siteList("traffic-matching-lists") }

type TrafficListGetCmd struct {
	ID string `arg:"" help:"Traffic-matching list id."`
}

func (c *TrafficListGetCmd) Run(rt *Runtime) error {
	return rt.siteGet("traffic-matching-lists/" + c.ID)
}

type TrafficListCreateCmd struct {
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *TrafficListCreateCmd) Run(rt *Runtime) error {
	return rt.configCreate("traffic-list create", "traffic-matching-lists", c.Data)
}

type TrafficListUpdateCmd struct {
	ID   string `arg:"" help:"Traffic-matching list id."`
	Data string `help:"Config body: path, '-' for stdin, or inline JSON." required:""`
}

func (c *TrafficListUpdateCmd) Run(rt *Runtime) error {
	return rt.configUpdate("traffic-list update", "traffic-matching-lists/"+c.ID, c.ID, c.Data)
}

type TrafficListDeleteCmd struct {
	ID string `arg:"" help:"Traffic-matching list id."`
}

func (c *TrafficListDeleteCmd) Run(rt *Runtime) error {
	return rt.configDelete("traffic-list delete", "traffic-matching-lists/"+c.ID, c.ID)
}

// --- config write helpers (build the plan body, then configWrite) -----------

func (rt *Runtime) configCreate(op, subpath, data string) error {
	body, err := rt.readData(data)
	if err != nil {
		return err
	}
	return rt.configWrite(op, http.MethodPost, subpath, body, map[string]any{"body": json.RawMessage(body)})
}

func (rt *Runtime) configUpdate(op, subpath, id, data string) error {
	body, err := rt.readData(data)
	if err != nil {
		return err
	}
	return rt.configWrite(op, http.MethodPut, subpath, body, map[string]any{"id": id, "body": json.RawMessage(body)})
}

func (rt *Runtime) configDelete(op, subpath, id string) error {
	return rt.configWrite(op, http.MethodDelete, subpath, nil, map[string]any{"id": id})
}

func (rt *Runtime) configReorder(op, subpath string, ids []string) error {
	body, _ := json.Marshal(ids)
	return rt.configWrite(op, http.MethodPut, subpath, body, map[string]any{"order": ids})
}

// --- apply ------------------------------------------------------------------

// ApplyCmd executes a previously previewed config plan by hash (reviewed-artifact, §2):
// load the plan, resolve the site, and issue exactly the persisted request.
type ApplyCmd struct {
	Hash string `arg:"" help:"Plan hash from a prior config --dry-run preview."`
}

func (c *ApplyCmd) Run(rt *Runtime) error {
	if err := rt.Guard("apply"); err != nil {
		return err
	}
	p, ok, err := plan.Load(c.Hash)
	if err != nil {
		return errs.New(errs.ExitConfig, "PLAN_READ_FAILED", err.Error(), "check $XDG_STATE_HOME/ufi/plans")
	}
	if !ok {
		return errs.New(errs.ExitUsage, "PLAN_NOT_FOUND", "no persisted plan for hash "+c.Hash,
			"re-run the config command with --dry-run to produce a plan")
	}
	if rt.Cfg.DryRun {
		return rt.Out.Emit(map[string]any{"dry_run": true, "hash": p.Hash, "op": p.Op, "method": p.Method, "path": p.Path, "plan": p.Summary})
	}
	cl, err := rt.local()
	if err != nil {
		return err
	}
	ctx := rt.ctx()
	site, err := rt.resolveSite(ctx, cl)
	if err != nil {
		return err
	}
	var bodyBytes []byte
	if len(p.Body) > 0 {
		bodyBytes = p.Body
	}
	v, err := cl.Send(ctx, p.Method, "/sites/"+site+"/"+p.Path, bodyBytes)
	if err != nil {
		return err
	}
	out := map[string]any{"ok": true, "hash": p.Hash, "op": p.Op}
	if v != nil {
		out["result"] = v
	}
	return rt.Out.Emit(out)
}

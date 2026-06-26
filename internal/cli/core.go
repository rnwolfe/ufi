package cli

// Core read-first nouns + low-stakes actions beyond device (info, site, client, wifi, voucher).
// Reads emit placeholder envelopes/objects until cli-implement wires the real UniFi client;
// mutations are gated and preview under --dry-run.

// --- info -------------------------------------------------------------------

type InfoCmd struct{}

func (c *InfoCmd) Run(rt *Runtime) error {
	return rt.emitPlaceholderObject() // GET /info → {application_version, capabilities}
}

// --- site -------------------------------------------------------------------

type SiteCmd struct {
	List SiteListCmd `cmd:"" help:"List sites on the console."`
}

type SiteListCmd struct{}

func (c *SiteListCmd) Run(rt *Runtime) error { return rt.emitEmptyList() }

// --- client -----------------------------------------------------------------

type ClientCmd struct {
	List        ClientListCmd        `cmd:"" help:"List connected clients."`
	Get         ClientGetCmd         `cmd:"" help:"Get one client by id."`
	Authorize   ClientAuthorizeCmd   `cmd:"" help:"Authorize a guest client (mutation)."`
	Unauthorize ClientUnauthorizeCmd `cmd:"" help:"Revoke a guest client's access (mutation)."`
}

type ClientListCmd struct{}

func (c *ClientListCmd) Run(rt *Runtime) error { return rt.emitEmptyList() }

type ClientGetCmd struct {
	ID string `arg:"" help:"Client id."`
}

func (c *ClientGetCmd) Run(rt *Runtime) error { return rt.emitPlaceholderObject() }

type ClientAuthorizeCmd struct {
	ID      string `arg:"" help:"Client id."`
	Minutes int    `help:"Time limit in minutes."`
	DataMB  int    `name:"data-mb" help:"Data usage limit in MB."`
	RxKbps  int    `name:"rx-kbps" help:"Download rate limit (kbps)."`
	TxKbps  int    `name:"tx-kbps" help:"Upload rate limit (kbps)."`
}

func (c *ClientAuthorizeCmd) Run(rt *Runtime) error {
	target := map[string]any{"id": c.ID}
	if c.Minutes > 0 {
		target["minutes"] = c.Minutes
	}
	if c.DataMB > 0 {
		target["data_mb"] = c.DataMB
	}
	if c.RxKbps > 0 {
		target["rx_kbps"] = c.RxKbps
	}
	if c.TxKbps > 0 {
		target["tx_kbps"] = c.TxKbps
	}
	return rt.gatedAction("client authorize", "AUTHORIZE_GUEST_ACCESS", target)
}

type ClientUnauthorizeCmd struct {
	ID string `arg:"" help:"Client id."`
}

func (c *ClientUnauthorizeCmd) Run(rt *Runtime) error {
	return rt.gatedAction("client unauthorize", "UNAUTHORIZE_GUEST_ACCESS", map[string]any{"id": c.ID})
}

// --- wifi -------------------------------------------------------------------

type WifiCmd struct {
	List WifiListCmd `cmd:"" help:"List WiFi broadcasts (SSIDs)."`
	Get  WifiGetCmd  `cmd:"" help:"Get one SSID by id."`
}

type WifiListCmd struct{}

func (c *WifiListCmd) Run(rt *Runtime) error { return rt.emitEmptyList() }

type WifiGetCmd struct {
	ID string `arg:"" help:"WiFi broadcast id."`
}

func (c *WifiGetCmd) Run(rt *Runtime) error { return rt.emitPlaceholderObject() }

// --- voucher ----------------------------------------------------------------

type VoucherCmd struct {
	List   VoucherListCmd   `cmd:"" help:"List hotspot vouchers."`
	Create VoucherCreateCmd `cmd:"" help:"Generate voucher(s) (mutation)."`
	Delete VoucherDeleteCmd `cmd:"" help:"Delete a voucher by id (mutation, idempotent)."`
}

type VoucherListCmd struct{}

func (c *VoucherListCmd) Run(rt *Runtime) error { return rt.emitEmptyList() }

type VoucherCreateCmd struct {
	Count   int    `default:"1" help:"How many vouchers to generate."`
	Minutes int    `help:"Time limit per voucher (minutes)."`
	Note    string `help:"Voucher note."`
}

func (c *VoucherCreateCmd) Run(rt *Runtime) error {
	target := map[string]any{"count": c.Count}
	if c.Minutes > 0 {
		target["minutes"] = c.Minutes
	}
	if c.Note != "" {
		target["note"] = c.Note
	}
	return rt.gatedAction("voucher create", "voucher.create", target)
}

type VoucherDeleteCmd struct {
	ID string `arg:"" help:"Voucher id."`
}

func (c *VoucherDeleteCmd) Run(rt *Runtime) error {
	return rt.idempotentDelete("voucher delete", "voucher", c.ID)
}

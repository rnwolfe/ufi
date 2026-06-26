package cli

// Core read-first nouns + low-stakes actions beyond device (info, site, client, wifi, voucher).

// --- info -------------------------------------------------------------------

type InfoCmd struct{}

func (c *InfoCmd) Run(rt *Runtime) error {
	cl, err := rt.local()
	if err != nil {
		return err
	}
	v, err := cl.GetObject(rt.ctx(), "/info")
	if err != nil {
		return err
	}
	return rt.Out.Emit(v)
}

// --- site -------------------------------------------------------------------

type SiteCmd struct {
	List SiteListCmd `cmd:"" help:"List sites on the console."`
}

type SiteListCmd struct{}

func (c *SiteListCmd) Run(rt *Runtime) error {
	cl, err := rt.local()
	if err != nil {
		return err
	}
	res, err := cl.List(rt.ctx(), "/sites", rt.listOpts())
	if err != nil {
		return err
	}
	return rt.emitList(res)
}

// --- client -----------------------------------------------------------------

type ClientCmd struct {
	List        ClientListCmd        `cmd:"" help:"List connected clients."`
	Get         ClientGetCmd         `cmd:"" help:"Get one client by id."`
	Authorize   ClientAuthorizeCmd   `cmd:"" help:"Authorize a guest client (mutation)."`
	Unauthorize ClientUnauthorizeCmd `cmd:"" help:"Revoke a guest client's access (mutation)."`
}

type ClientListCmd struct{}

func (c *ClientListCmd) Run(rt *Runtime) error {
	return rt.siteList("clients", "name", "hostname", "note")
}

type ClientGetCmd struct {
	ID string `arg:"" help:"Client id."`
}

func (c *ClientGetCmd) Run(rt *Runtime) error {
	return rt.siteGet("clients/"+c.ID, "name", "hostname", "note")
}

type ClientAuthorizeCmd struct {
	ID      string `arg:"" help:"Client id."`
	Minutes int    `help:"Time limit in minutes (timeLimitMinutes)."`
	DataMB  int    `name:"data-mb" help:"Data usage limit in MB (dataUsageLimitMBytes)."`
	RxKbps  int    `name:"rx-kbps" help:"Download rate limit, kbps (rxRateLimitKbps)."`
	TxKbps  int    `name:"tx-kbps" help:"Upload rate limit, kbps (txRateLimitKbps)."`
}

func (c *ClientAuthorizeCmd) Run(rt *Runtime) error {
	body := map[string]any{"action": "AUTHORIZE_GUEST_ACCESS"}
	preview := map[string]any{"action": "AUTHORIZE_GUEST_ACCESS", "id": c.ID}
	if c.Minutes > 0 {
		body["timeLimitMinutes"] = c.Minutes
		preview["minutes"] = c.Minutes
	}
	if c.DataMB > 0 {
		body["dataUsageLimitMBytes"] = c.DataMB
	}
	if c.RxKbps > 0 {
		body["rxRateLimitKbps"] = c.RxKbps
	}
	if c.TxKbps > 0 {
		body["txRateLimitKbps"] = c.TxKbps
	}
	return rt.siteAction("client authorize", "clients/"+c.ID+"/actions", body, preview)
}

type ClientUnauthorizeCmd struct {
	ID string `arg:"" help:"Client id."`
}

func (c *ClientUnauthorizeCmd) Run(rt *Runtime) error {
	return rt.siteAction("client unauthorize", "clients/"+c.ID+"/actions",
		map[string]any{"action": "UNAUTHORIZE_GUEST_ACCESS"},
		map[string]any{"action": "UNAUTHORIZE_GUEST_ACCESS", "id": c.ID})
}

// --- wifi -------------------------------------------------------------------

type WifiCmd struct {
	List WifiListCmd `cmd:"" help:"List WiFi broadcasts (SSIDs)."`
	Get  WifiGetCmd  `cmd:"" help:"Get one SSID by id."`
}

type WifiListCmd struct{}

func (c *WifiListCmd) Run(rt *Runtime) error { return rt.siteList("wifi/broadcasts", "name") }

type WifiGetCmd struct {
	ID string `arg:"" help:"WiFi broadcast id."`
}

func (c *WifiGetCmd) Run(rt *Runtime) error { return rt.siteGet("wifi/broadcasts/"+c.ID, "name") }

// --- voucher ----------------------------------------------------------------

type VoucherCmd struct {
	List   VoucherListCmd   `cmd:"" help:"List hotspot vouchers."`
	Create VoucherCreateCmd `cmd:"" help:"Generate voucher(s) (mutation)."`
	Delete VoucherDeleteCmd `cmd:"" help:"Delete a voucher by id (mutation, idempotent)."`
}

type VoucherListCmd struct{}

func (c *VoucherListCmd) Run(rt *Runtime) error {
	return rt.siteList("hotspot/vouchers", "name", "note")
}

// VoucherCreateCmd — the API requires name + timeLimitMinutes.
type VoucherCreateCmd struct {
	Name    string `arg:"" help:"Voucher note/name (required by the API)."`
	Minutes int    `required:"" help:"Time limit per voucher in minutes (timeLimitMinutes, required)."`
	Count   int    `default:"1" help:"How many vouchers to generate."`
	Guests  int    `help:"Authorized guests per voucher (authorizedGuestLimit)."`
	DataMB  int    `name:"data-mb" help:"Data usage limit in MB (dataUsageLimitMBytes)."`
}

func (c *VoucherCreateCmd) Run(rt *Runtime) error {
	body := map[string]any{"name": c.Name, "timeLimitMinutes": c.Minutes, "count": c.Count}
	if c.Guests > 0 {
		body["authorizedGuestLimit"] = c.Guests
	}
	if c.DataMB > 0 {
		body["dataUsageLimitMBytes"] = c.DataMB
	}
	preview := map[string]any{"action": "voucher.create", "name": c.Name, "minutes": c.Minutes, "count": c.Count}
	return rt.siteAction("voucher create", "hotspot/vouchers", body, preview)
}

type VoucherDeleteCmd struct {
	ID string `arg:"" help:"Voucher id."`
}

func (c *VoucherDeleteCmd) Run(rt *Runtime) error {
	return rt.siteDelete("voucher delete", "hotspot/vouchers/"+c.ID, "voucher", c.ID)
}

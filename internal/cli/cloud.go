package cli

// Cloud fleet reads via the Site Manager API (api.ui.com, X-API-KEY from unifi.ui.com).
// All reads; cli-implement wires the cloud client + the 10,000 req/min Retry-After handling.

type CloudCmd struct {
	Host       CloudHostCmd       `cmd:"" help:"List UniFi OS hosts on the account."`
	Site       CloudSiteCmd       `cmd:"" help:"List sites across all hosts."`
	Device     CloudDeviceCmd     `cmd:"" help:"List devices across all sites."`
	IspMetrics CloudIspMetricsCmd `cmd:"" name:"isp-metrics" help:"Internet-health / ISP metrics."`
}

type CloudHostCmd struct {
	List CloudHostListCmd `cmd:"" help:"List hosts."`
}

type CloudHostListCmd struct{}

func (c *CloudHostListCmd) Run(rt *Runtime) error { return rt.emitEmptyList() }

type CloudSiteCmd struct {
	List CloudSiteListCmd `cmd:"" help:"List sites across hosts."`
}

type CloudSiteListCmd struct{}

func (c *CloudSiteListCmd) Run(rt *Runtime) error { return rt.emitEmptyList() }

type CloudDeviceCmd struct {
	List CloudDeviceListCmd `cmd:"" help:"List devices across sites."`
}

type CloudDeviceListCmd struct{}

func (c *CloudDeviceListCmd) Run(rt *Runtime) error { return rt.emitEmptyList() }

type CloudIspMetricsCmd struct{}

func (c *CloudIspMetricsCmd) Run(rt *Runtime) error { return rt.emitEmptyList() }

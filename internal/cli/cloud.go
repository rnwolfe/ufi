package cli

// Cloud fleet reads via the Site Manager API (api.ui.com, X-API-KEY from unifi.ui.com).
// All reads. NOTE: these paths follow spec.md but are not yet validated against a live cloud
// account (no cloud key on hand at implement time); they degrade to a clean API error if a path
// is wrong. The Site Manager response is unwrapped to its `data` collection where present.

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

func (c *CloudHostListCmd) Run(rt *Runtime) error { return rt.cloudList("/hosts") }

type CloudSiteCmd struct {
	List CloudSiteListCmd `cmd:"" help:"List sites across hosts."`
}
type CloudSiteListCmd struct{}

func (c *CloudSiteListCmd) Run(rt *Runtime) error { return rt.cloudList("/sites") }

type CloudDeviceCmd struct {
	List CloudDeviceListCmd `cmd:"" help:"List devices across sites."`
}
type CloudDeviceListCmd struct{}

func (c *CloudDeviceListCmd) Run(rt *Runtime) error { return rt.cloudList("/devices") }

type CloudIspMetricsCmd struct{}

func (c *CloudIspMetricsCmd) Run(rt *Runtime) error { return rt.cloudList("/isp-metrics/5m") }

// cloudList GETs a Site Manager path and emits its `data` collection in the list envelope.
func (rt *Runtime) cloudList(path string) error {
	cl, err := rt.cloud()
	if err != nil {
		return err
	}
	v, err := cl.GetObject(rt.ctx(), path)
	if err != nil {
		return err
	}
	if m, ok := v.(map[string]any); ok {
		if data, ok := m["data"].([]any); ok {
			return rt.Out.Emit(listEnvelope(data, len(data), nil))
		}
	}
	return rt.Out.Emit(v)
}

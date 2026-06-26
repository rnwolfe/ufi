package cli

// DeviceCmd groups device subcommands (local Integration API: /sites/{id}/devices).
type DeviceCmd struct {
	List      DeviceListCmd      `cmd:"" help:"List adopted devices for a site."`
	Get       DeviceGetCmd       `cmd:"" help:"Get one device by id."`
	Stats     DeviceStatsCmd     `cmd:"" help:"Latest device statistics."`
	Restart   DeviceRestartCmd   `cmd:"" help:"Restart a device (mutation)."`
	PortCycle DevicePortCycleCmd `cmd:"" name:"port-cycle" help:"Power-cycle a switch port (mutation)."`
}

type DeviceListCmd struct{}

func (c *DeviceListCmd) Run(rt *Runtime) error { return rt.siteList("devices", "name") }

type DeviceGetCmd struct {
	ID string `arg:"" help:"Device id."`
}

func (c *DeviceGetCmd) Run(rt *Runtime) error { return rt.siteGet("devices/"+c.ID, "name") }

type DeviceStatsCmd struct {
	ID string `arg:"" help:"Device id."`
}

func (c *DeviceStatsCmd) Run(rt *Runtime) error {
	return rt.siteGet("devices/" + c.ID + "/statistics/latest")
}

type DeviceRestartCmd struct {
	ID string `arg:"" help:"Device id."`
}

func (c *DeviceRestartCmd) Run(rt *Runtime) error {
	return rt.siteAction("device restart", "devices/"+c.ID+"/actions",
		map[string]any{"action": "RESTART"},
		map[string]any{"action": "RESTART", "id": c.ID})
}

type DevicePortCycleCmd struct {
	ID   string `arg:"" help:"Device id."`
	Port int    `arg:"" help:"Switch port index (portIdx)."`
}

func (c *DevicePortCycleCmd) Run(rt *Runtime) error {
	return rt.siteAction("device port-cycle", "devices/"+c.ID+"/interfaces/ports/"+itoa(c.Port)+"/actions",
		map[string]any{"action": "POWER_CYCLE"},
		map[string]any{"action": "POWER_CYCLE", "id": c.ID, "port": c.Port})
}

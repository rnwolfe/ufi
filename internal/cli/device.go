package cli

import "github.com/rnwolfe/ufi/internal/errs"

// DeviceCmd groups device subcommands (local Integration API: /sites/{id}/devices).
type DeviceCmd struct {
	List      DeviceListCmd      `cmd:"" help:"List adopted devices for a site."`
	Get       DeviceGetCmd       `cmd:"" help:"Get one device by id."`
	Stats     DeviceStatsCmd     `cmd:"" help:"Latest device statistics."`
	Restart   DeviceRestartCmd   `cmd:"" help:"Restart a device (mutation)."`
	PortCycle DevicePortCycleCmd `cmd:"" name:"port-cycle" help:"Power-cycle a switch port (mutation)."`
}

type DeviceListCmd struct{}

// Run lists devices. Backed by the placeholder store until cli-implement wires the real
// GET /sites/{id}/devices call; returns the stable list envelope either way.
func (c *DeviceListCmd) Run(rt *Runtime) error {
	devs, err := rt.Store.List()
	if err != nil {
		return errs.New(errs.ExitConfig, "STORE_ERROR", err.Error(), "check the store path / UFI_STORE")
	}
	return rt.Out.Emit(listEnvelope(devs, len(devs), nil))
}

type DeviceGetCmd struct {
	ID string `arg:"" help:"Device id."`
}

func (c *DeviceGetCmd) Run(rt *Runtime) error {
	d, ok, err := rt.Store.Get(c.ID)
	if err != nil {
		return errs.New(errs.ExitConfig, "STORE_ERROR", err.Error(), "check the store path")
	}
	if !ok {
		return errs.NotFound("device", c.ID)
	}
	return rt.Out.Emit(d)
}

type DeviceStatsCmd struct {
	ID string `arg:"" help:"Device id."`
}

func (c *DeviceStatsCmd) Run(rt *Runtime) error {
	return rt.emitPlaceholderObject()
}

type DeviceRestartCmd struct {
	ID string `arg:"" help:"Device id."`
}

func (c *DeviceRestartCmd) Run(rt *Runtime) error {
	return rt.gatedAction("device restart", "RESTART", map[string]any{"id": c.ID})
}

type DevicePortCycleCmd struct {
	ID   string `arg:"" help:"Device id."`
	Port int    `arg:"" help:"Switch port index."`
}

func (c *DevicePortCycleCmd) Run(rt *Runtime) error {
	return rt.gatedAction("device port-cycle", "POWER_CYCLE", map[string]any{"id": c.ID, "port": c.Port})
}

package cli

import "github.com/rnwolfe/ufi/internal/errs"

// CloudCmd is a hidden placeholder. The Site Manager (cloud) surface is intentionally not
// shipped in this build — ufi is local-Integration-API only for now. The command stays parseable
// (so `ufi cloud …` gets a clear pointer instead of a "did you mean") but is hidden from help and
// schema. Re-enable by restoring the read commands once a cloud key path is validated.
type CloudCmd struct {
	Args []string `arg:"" optional:"" help:"(unavailable)"`
}

func (c *CloudCmd) Run(rt *Runtime) error {
	return errs.New(errs.ExitUnsupported, "UNSUPPORTED",
		"the Site Manager (cloud) surface is not available in this build — ufi is local-only for now",
		"if you want cloud (api.ui.com) support, please open an issue: https://github.com/rnwolfe/ufi/issues")
}

package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"

	"github.com/rnwolfe/ufi/internal/errs"
)

// schemaVersion is the top-level version stamped on every list envelope (spec: Output schema).
// Bump only for breaking output changes; field additions are append-only and don't bump it.
const schemaVersion = 1

// listEnvelope wraps items in the stable list envelope:
// { schemaVersion, items, count, nextCursor }. nextCursor is nil at end-of-results.
func listEnvelope(items any, count int, nextCursor any) map[string]any {
	if items == nil {
		items = []any{}
	}
	return map[string]any{
		"schemaVersion": schemaVersion,
		"items":         items,
		"count":         count,
		"nextCursor":    nextCursor,
	}
}

// emitEmptyList emits an empty list envelope. Placeholder reads use this until cli-implement
// repoints them at the real UniFi client. Honest: a real empty result, not a fake.
func (rt *Runtime) emitEmptyList() error {
	return rt.Out.Emit(listEnvelope([]any{}, 0, nil))
}

// emitPlaceholderObject emits an empty object for single-resource reads not yet wired.
func (rt *Runtime) emitPlaceholderObject() error {
	return rt.Out.Emit(map[string]any{})
}

// gatedAction runs a low-stakes single-target action (device/client/voucher actions):
// gate first (contract §2), then --dry-run prints the plan, otherwise the real upstream call
// (wired by cli-implement) would fire. The skeleton returns NOT_IMPLEMENTED on real execution
// rather than faking success — preview the gate with --dry-run.
func (rt *Runtime) gatedAction(op, action string, target map[string]any) error {
	if err := rt.Guard(op); err != nil {
		return err
	}
	plan := map[string]any{"action": action}
	for k, v := range target {
		plan[k] = v
	}
	if rt.Cfg.DryRun {
		plan["dry_run"] = true
		return rt.Out.Emit(plan)
	}
	return errs.New(errs.ExitUnsupported, "NOT_IMPLEMENTED",
		op+" is not yet wired to the UniFi API",
		"the upstream call is implemented by cli-implement; run with --dry-run to preview the gate")
}

// idempotentDelete models a delete that soft-succeeds when the target is already gone
// (contract §9). The placeholder has no state, so it reports existed:false.
func (rt *Runtime) idempotentDelete(op, kind, id string) error {
	if err := rt.Guard(op); err != nil {
		return err
	}
	if rt.Cfg.DryRun {
		return rt.Out.Emit(map[string]any{"dry_run": true, "action": op, "id": id})
	}
	return rt.Out.Emit(map[string]any{"ok": true, "kind": kind, "id": id, "existed": false})
}

// previewConfig implements the reviewed-artifact path for declarative config writes
// (contract §2): compute a deterministic plan hash and emit the plan for review. The persist
// + `ufi apply <hash>` execution is wired by cli-implement; here we only produce the preview.
func (rt *Runtime) previewConfig(op string, plan map[string]any) error {
	if err := rt.Guard(op); err != nil {
		return err
	}
	h := planHash(op, plan)
	return rt.Out.Emit(map[string]any{
		"action":  op,
		"plan":    plan,
		"hash":    h,
		"dry_run": true,
		"note":    "preview only — run `ufi apply " + h + " --allow-mutations` to execute (wired by cli-implement)",
	})
}

func planHash(op string, plan map[string]any) string {
	b, _ := json.Marshal(map[string]any{"op": op, "plan": plan})
	sum := sha256.Sum256(b)
	return hex.EncodeToString(sum[:])[:12]
}

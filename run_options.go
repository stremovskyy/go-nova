package go_nova

import (
	"encoding/json"
	"fmt"

	"github.com/stremovskyy/go-nova/log"
)

// RunOption controls behavior of a single SDK call.
type RunOption func(*runOptions)

// DryRunHandler receives information about a skipped request.
type DryRunHandler func(method string, url string, payload any)

type runOptions struct {
	dryRun       bool
	dryRunHandle DryRunHandler
}

var dryRunLogger = log.NewDefault()

// DryRun skips the underlying HTTP call.
//
// Optional handler lets you inspect the request payload.
func DryRun(handler ...DryRunHandler) RunOption {
	return func(o *runOptions) {
		o.dryRun = true
		if len(handler) > 0 && handler[0] != nil {
			o.dryRunHandle = handler[0]
			return
		}
		o.dryRunHandle = defaultDryRunHandler
	}
}

func collectRunOptions(opts []RunOption) *runOptions {
	if len(opts) == 0 {
		return nil
	}

	r := &runOptions{}
	for _, opt := range opts {
		if opt != nil {
			opt(r)
		}
	}
	return r
}

func (o *runOptions) isDryRun() bool {
	return o != nil && o.dryRun
}

func (o *runOptions) handleDryRun(method string, url string, payload any) {
	if o == nil || !o.dryRun || o.dryRunHandle == nil {
		return
	}
	o.dryRunHandle(method, url, payload)
}

func shouldDryRun(runOpts []RunOption, method string, url string, payload any) bool {
	opts := collectRunOptions(runOpts)
	if !opts.isDryRun() {
		return false
	}
	opts.handleDryRun(method, url, payload)
	return true
}

func defaultDryRunHandler(method string, url string, payload any) {
	dryRunLogger.Infof("Dry run: skipping request %s %s", method, url)
	if payload == nil {
		dryRunLogger.Infof("Dry run payload: <nil>")
		return
	}
	if b, ok := payload.([]byte); ok {
		dryRunLogger.Infof("Dry run payload:\n%s", string(b))
		return
	}
	if s, ok := payload.(string); ok {
		dryRunLogger.Infof("Dry run payload:\n%s", s)
		return
	}
	dryRunLogger.Infof("Dry run payload:\n%s", marshalIndent(payload))
}

func marshalIndent(v any) string {
	if v == nil {
		return "<nil>"
	}
	out, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return fmt.Sprintf("unable to marshal %T: %v", v, err)
	}
	return string(out)
}

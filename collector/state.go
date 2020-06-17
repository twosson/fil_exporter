package collector

import "github.com/go-kit/kit/log"

var (
	collectorState   = make(map[string]*bool)
	factories        = make(map[string]func(logger log.Logger) (Collector, error))
	forcedCollectors = map[string]bool{}
)

const (
	Namespace       = "fil"

	DefaultEnabled  = true
	DefaultDisabled = false
)
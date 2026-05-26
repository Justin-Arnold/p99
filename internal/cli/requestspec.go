package cli

import (
	"flag"
	"fmt"
	"time"

	"github.com/Justin-Arnold/p99/internal/probe"
	"github.com/Justin-Arnold/p99/internal/requestspec"
)

func explicitFlags(fs *flag.FlagSet) map[string]bool {
	out := map[string]bool{}
	fs.Visit(func(f *flag.Flag) {
		out[f.Name] = true
	})
	return out
}

func loadRequestPlan(path, baseURL string, seed int64, seedProvided bool, cfg *probe.HTTPConfig) (*requestspec.Plan, error) {
	plan, err := requestspec.Load(path, baseURL)
	if err != nil {
		return nil, err
	}
	cfg.RequestSpec = path
	cfg.RequestBaseURL = plan.BaseURL
	if seedProvided {
		cfg.RequestSeed = seed
	} else {
		cfg.RequestSeed = time.Now().UnixNano()
	}
	return &plan, nil
}

func rejectRequestSpecConflicts(fs *flag.FlagSet, names ...string) error {
	explicit := explicitFlags(fs)
	for _, name := range names {
		if explicit[name] {
			return fmt.Errorf("--request-spec cannot be used with --%s", name)
		}
	}
	return nil
}

func rejectSpecOnlyFlags(fs *flag.FlagSet, specPath string, names ...string) error {
	if specPath != "" {
		return nil
	}
	explicit := explicitFlags(fs)
	for _, name := range names {
		if explicit[name] {
			return fmt.Errorf("--%s requires --request-spec", name)
		}
	}
	return nil
}

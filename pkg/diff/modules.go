package diff

import (
	"sort"

	"github.com/Controlwatch/control-watch/pkg/model"
)

const modulesSubject = "compliance.modules"

func measuredModules(observation model.Observation) model.Measured[[]model.Module] {
	if observation.Compliance == nil {
		return model.Measure([]model.Module{})
	}
	return observation.Compliance.Modules
}

func modulesByAddress(modules []model.Module) map[string]model.Module {
	indexed := make(map[string]model.Module, len(modules))
	for _, module := range modules {
		indexed[module.Address] = module
	}
	return indexed
}

func sortedModuleAddresses(modules map[string]model.Module) []string {
	addresses := make([]string, 0, len(modules))
	for address := range modules {
		addresses = append(addresses, address)
	}
	sort.Strings(addresses)
	return addresses
}

package plugin

import (
	"context"
	"time"

	"github.com/kubeflow/model-registry/pkg/tenancy"
)

// pluginRefreshAdapter wraps a RefreshProvider to satisfy the jobs.PluginRefresher
// interface. It also persists refresh status after each refresh completes.
type pluginRefreshAdapter struct {
	rp         RefreshProvider
	srv        *Server
	pluginName string
}

func (a *pluginRefreshAdapter) Refresh(ctx context.Context, sourceID string) (entitiesLoaded, entitiesRemoved int, duration time.Duration, err error) {
	result, err := a.rp.Refresh(ctx, sourceID)
	if err != nil {
		return 0, 0, 0, err
	}

	// Persist refresh status.
	if a.srv != nil && result != nil {
		ns := tenancy.NamespaceFromContext(ctx)
		a.srv.saveRefreshStatus(ns, a.pluginName, sourceID, result)
	}

	if result == nil {
		return 0, 0, 0, nil
	}

	return result.EntitiesLoaded, result.EntitiesRemoved, result.Duration, nil
}

func (a *pluginRefreshAdapter) RefreshAll(ctx context.Context) (entitiesLoaded, entitiesRemoved int, duration time.Duration, err error) {
	result, err := a.rp.RefreshAll(ctx)
	if err != nil {
		return 0, 0, 0, err
	}

	// Persist refresh status.
	if a.srv != nil && result != nil {
		ns := tenancy.NamespaceFromContext(ctx)
		a.srv.saveRefreshStatus(ns, a.pluginName, "_all", result)
	}

	if result == nil {
		return 0, 0, 0, nil
	}

	return result.EntitiesLoaded, result.EntitiesRemoved, result.Duration, nil
}

package repositories

import (
	"encoding/json"
	"fmt"
	"io"
	"net/url"

	"github.com/kubeflow/model-registry/ui/bff/internal/integrations/httpclient"
)

const (
	governanceBasePath          = "/api/governance/v1alpha1"
	governanceAssetPathFmt      = governanceBasePath + "/assets/%s/%s/%s"
	governanceAssetHistoryFmt   = governanceAssetPathFmt + "/history"
	governanceAssetActionFmt    = governanceAssetPathFmt + "/actions/%s"
	governanceAssetVersionsFmt  = governanceAssetPathFmt + "/versions"
	governanceAssetBindingsFmt  = governanceAssetPathFmt + "/bindings"
	governanceAssetBindingFmt   = governanceAssetBindingsFmt + "/%s"
	governanceApprovalsPath     = governanceBasePath + "/approvals"
	governanceApprovalPathFmt   = governanceApprovalsPath + "/%s"
	governanceApprovalDecFmt    = governanceApprovalPathFmt + "/decisions"
	governanceApprovalCancelFmt = governanceApprovalPathFmt + "/cancel"
	governancePoliciesPath      = governanceBasePath + "/policies"
)

// CatalogGovernanceInterface defines methods for governance API proxying.
type CatalogGovernanceInterface interface {
	GetGovernance(client httpclient.HTTPClientInterface, plugin, kind, name string) (json.RawMessage, error)
	PatchGovernance(client httpclient.HTTPClientInterface, plugin, kind, name string, body io.Reader) (json.RawMessage, error)
	GetGovernanceHistory(client httpclient.HTTPClientInterface, plugin, kind, name string, queryParams url.Values) (json.RawMessage, error)
	PostGovernanceAction(client httpclient.HTTPClientInterface, plugin, kind, name, action string, body io.Reader) (json.RawMessage, error)
	ListVersions(client httpclient.HTTPClientInterface, plugin, kind, name string, queryParams url.Values) (json.RawMessage, error)
	CreateVersion(client httpclient.HTTPClientInterface, plugin, kind, name string, body io.Reader) (json.RawMessage, error)
	ListBindings(client httpclient.HTTPClientInterface, plugin, kind, name string) (json.RawMessage, error)
	SetBinding(client httpclient.HTTPClientInterface, plugin, kind, name, env string, body io.Reader) (json.RawMessage, error)
	ListApprovals(client httpclient.HTTPClientInterface, queryParams url.Values) (json.RawMessage, error)
	GetApproval(client httpclient.HTTPClientInterface, id string) (json.RawMessage, error)
	PostApprovalDecision(client httpclient.HTTPClientInterface, id string, body io.Reader) (json.RawMessage, error)
	CancelApproval(client httpclient.HTTPClientInterface, id string, body io.Reader) (json.RawMessage, error)
	ListPolicies(client httpclient.HTTPClientInterface) (json.RawMessage, error)
}

// CatalogGovernance implements CatalogGovernanceInterface.
type CatalogGovernance struct {
	CatalogGovernanceInterface
}

func (g CatalogGovernance) GetGovernance(client httpclient.HTTPClientInterface, plugin, kind, name string) (json.RawMessage, error) {
	path := fmt.Sprintf(governanceAssetPathFmt, plugin, kind, name)
	data, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error fetching governance: %w", err)
	}
	return json.RawMessage(data), nil
}

func (g CatalogGovernance) PatchGovernance(client httpclient.HTTPClientInterface, plugin, kind, name string, body io.Reader) (json.RawMessage, error) {
	path := fmt.Sprintf(governanceAssetPathFmt, plugin, kind, name)
	data, err := client.PATCH(path, body)
	if err != nil {
		return nil, fmt.Errorf("error patching governance: %w", err)
	}
	return json.RawMessage(data), nil
}

func (g CatalogGovernance) GetGovernanceHistory(client httpclient.HTTPClientInterface, plugin, kind, name string, queryParams url.Values) (json.RawMessage, error) {
	path := fmt.Sprintf(governanceAssetHistoryFmt, plugin, kind, name)
	path = UrlWithPageParams(path, queryParams)
	data, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error fetching governance history: %w", err)
	}
	return json.RawMessage(data), nil
}

func (g CatalogGovernance) PostGovernanceAction(client httpclient.HTTPClientInterface, plugin, kind, name, action string, body io.Reader) (json.RawMessage, error) {
	path := fmt.Sprintf(governanceAssetActionFmt, plugin, kind, name, action)
	data, err := client.POST(path, body)
	if err != nil {
		return nil, fmt.Errorf("error executing governance action: %w", err)
	}
	return json.RawMessage(data), nil
}

func (g CatalogGovernance) ListVersions(client httpclient.HTTPClientInterface, plugin, kind, name string, queryParams url.Values) (json.RawMessage, error) {
	path := fmt.Sprintf(governanceAssetVersionsFmt, plugin, kind, name)
	path = UrlWithPageParams(path, queryParams)
	data, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error listing versions: %w", err)
	}
	return json.RawMessage(data), nil
}

func (g CatalogGovernance) CreateVersion(client httpclient.HTTPClientInterface, plugin, kind, name string, body io.Reader) (json.RawMessage, error) {
	path := fmt.Sprintf(governanceAssetVersionsFmt, plugin, kind, name)
	data, err := client.POST(path, body)
	if err != nil {
		return nil, fmt.Errorf("error creating version: %w", err)
	}
	return json.RawMessage(data), nil
}

func (g CatalogGovernance) ListBindings(client httpclient.HTTPClientInterface, plugin, kind, name string) (json.RawMessage, error) {
	path := fmt.Sprintf(governanceAssetBindingsFmt, plugin, kind, name)
	data, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error listing bindings: %w", err)
	}
	return json.RawMessage(data), nil
}

func (g CatalogGovernance) SetBinding(client httpclient.HTTPClientInterface, plugin, kind, name, env string, body io.Reader) (json.RawMessage, error) {
	// PUT is not available on the HTTP client; use PATCH to forward the binding payload.
	// The catalog server uses PUT semantics for set-binding, but the body content is the same.
	path := fmt.Sprintf(governanceAssetBindingFmt, plugin, kind, name, env)
	data, err := client.PATCH(path, body)
	if err != nil {
		return nil, fmt.Errorf("error setting binding: %w", err)
	}
	return json.RawMessage(data), nil
}

func (g CatalogGovernance) ListApprovals(client httpclient.HTTPClientInterface, queryParams url.Values) (json.RawMessage, error) {
	path := governanceApprovalsPath
	path = UrlWithPageParams(path, queryParams)
	data, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error listing approvals: %w", err)
	}
	return json.RawMessage(data), nil
}

func (g CatalogGovernance) GetApproval(client httpclient.HTTPClientInterface, id string) (json.RawMessage, error) {
	path := fmt.Sprintf(governanceApprovalPathFmt, id)
	data, err := client.GET(path)
	if err != nil {
		return nil, fmt.Errorf("error fetching approval: %w", err)
	}
	return json.RawMessage(data), nil
}

func (g CatalogGovernance) PostApprovalDecision(client httpclient.HTTPClientInterface, id string, body io.Reader) (json.RawMessage, error) {
	path := fmt.Sprintf(governanceApprovalDecFmt, id)
	data, err := client.POST(path, body)
	if err != nil {
		return nil, fmt.Errorf("error submitting approval decision: %w", err)
	}
	return json.RawMessage(data), nil
}

func (g CatalogGovernance) CancelApproval(client httpclient.HTTPClientInterface, id string, body io.Reader) (json.RawMessage, error) {
	path := fmt.Sprintf(governanceApprovalCancelFmt, id)
	data, err := client.POST(path, body)
	if err != nil {
		return nil, fmt.Errorf("error canceling approval: %w", err)
	}
	return json.RawMessage(data), nil
}

func (g CatalogGovernance) ListPolicies(client httpclient.HTTPClientInterface) (json.RawMessage, error) {
	data, err := client.GET(governancePoliciesPath)
	if err != nil {
		return nil, fmt.Errorf("error listing policies: %w", err)
	}
	return json.RawMessage(data), nil
}

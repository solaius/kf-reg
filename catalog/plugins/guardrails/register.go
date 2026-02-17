package guardrails

import "github.com/kubeflow/model-registry/pkg/catalog/plugin"

func init() {
	plugin.Register(&GuardrailPlugin{})
}

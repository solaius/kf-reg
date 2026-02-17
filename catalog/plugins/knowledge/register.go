package knowledge

import "github.com/kubeflow/model-registry/pkg/catalog/plugin"

func init() {
	plugin.Register(&KnowledgeSourcePlugin{})
}

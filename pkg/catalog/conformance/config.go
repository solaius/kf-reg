package conformance

// HarnessConfig configures the conformance test harness.
type HarnessConfig struct {
	// PluginName restricts testing to a specific plugin (empty = all).
	PluginName string

	// ServerURL is the base URL of the catalog server.
	ServerURL string

	// ExpectedEntityKinds lists the entity kinds expected for this plugin.
	ExpectedEntityKinds []string

	// ExpectedCaps describes what capabilities the plugin should have.
	ExpectedCaps ExpectedCaps

	// SkipCategories lists categories to skip: "capabilities", "list_get",
	// "sources", "security", "observability", "openapi".
	SkipCategories []string
}

// ExpectedCaps describes what capabilities the plugin should have.
type ExpectedCaps struct {
	HasSources bool
	HasActions bool
	HasRefresh bool
	HasUIHints bool
}

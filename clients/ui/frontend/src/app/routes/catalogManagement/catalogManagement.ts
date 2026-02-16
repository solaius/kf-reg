export const CATALOG_MANAGEMENT_PAGE_TITLE = 'Catalog Management';
export const CATALOG_MANAGEMENT_DESCRIPTION =
  'Manage plugins, sources, and diagnostics for all catalog asset types.';

export const catalogManagementUrl = (): string => '/catalog-management';

export const catalogPluginUrl = (pluginName: string): string =>
  `${catalogManagementUrl()}/plugin/${encodeURIComponent(pluginName)}`;

export const catalogPluginSourcesUrl = (pluginName: string): string =>
  `${catalogPluginUrl(pluginName)}/sources`;

export const catalogPluginDiagnosticsUrl = (pluginName: string): string =>
  `${catalogPluginUrl(pluginName)}/diagnostics`;

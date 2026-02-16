export const MCP_CATALOG_PAGE_TITLE = 'MCP Catalog';
export const MCP_CATALOG_DESCRIPTION =
  'Discover and manage MCP servers available for your AI applications.';

export const mcpCatalogUrl = (): string => '/mcp-catalog';
export const mcpServerDetailUrl = (name: string): string => `/mcp-catalog/${name}`;

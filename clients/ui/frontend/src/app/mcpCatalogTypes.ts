export type McpToolParameter = {
  name: string;
  type: string;
  description: string;
  required: boolean;
};

export type McpTool = {
  name: string;
  description: string;
  accessType: 'read_only' | 'read_write' | 'destructive';
  parameters?: McpToolParameter[];
};

export type McpResource = {
  name: string;
  description: string;
  uri?: string;
};

export type McpServer = {
  id?: string;
  name: string;
  description?: string;
  serverUrl: string;
  transportType?: string;
  deploymentMode?: string;
  image?: string;
  endpoint?: string;
  supportedTransports?: string;
  license?: string;
  verified?: boolean;
  certified?: boolean;
  provider?: string;
  logo?: string;
  category?: string;
  toolCount?: number;
  resourceCount?: number;
  promptCount?: number;
  tools?: McpTool[];
  resources?: McpResource[];
  readme?: string;
  sourceUrl?: string;
  version?: string;
  lastModified?: string;
  tags?: string[];
  sourceLabel?: string;
  customProperties?: Record<string, unknown>;
};

export type McpServerList = {
  items: McpServer[];
  size: number;
  nextPageToken?: string;
};

export enum McpCatalogFilterKey {
  DEPLOYMENT_MODE = 'deploymentMode',
  CATEGORY = 'category',
  LICENSE = 'license',
  TRANSPORT = 'transportType',
}

export type McpCatalogFilterStates = {
  [McpCatalogFilterKey.DEPLOYMENT_MODE]: string[];
  [McpCatalogFilterKey.CATEGORY]: string[];
  [McpCatalogFilterKey.LICENSE]: string[];
  [McpCatalogFilterKey.TRANSPORT]: string[];
};

export const MCP_ALL_CATEGORIES = 'All servers';
export const MCP_OTHER_SERVERS = '__other__';
export const MCP_OTHER_SERVERS_DISPLAY = 'Other servers';

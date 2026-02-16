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

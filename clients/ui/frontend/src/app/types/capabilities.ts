/** V2 Capabilities types mirroring the catalog server schema. */

export type V2ColumnHint = {
  name: string;
  displayName: string;
  path: string;
  type?: string;
  sortable?: boolean;
  width?: string;
};

export type V2FilterField = {
  name: string;
  displayName: string;
  type: 'text' | 'select' | 'boolean' | 'number';
  options?: string[];
  operators?: string[];
};

export type V2FieldHint = {
  name: string;
  displayName: string;
  path: string;
  type?: string;
  section?: string;
};

export type ActionParameter = {
  name: string;
  type: 'string' | 'boolean' | 'number' | 'tags' | 'key-value';
  label: string;
  required?: boolean;
  description?: string;
  defaultValue?: unknown;
};

export type ActionDefinition = {
  id: string;
  displayName: string;
  description?: string;
  scope?: string;
  supportsDryRun?: boolean;
  idempotent?: boolean;
  destructive?: boolean;
  parameters?: ActionParameter[];
};

export type EntityCapabilities = {
  kind: string;
  plural: string;
  displayName: string;
  description?: string;
  endpoints: {
    list: string;
    get: string;
    action?: string;
  };
  fields: {
    columns: V2ColumnHint[];
    filterFields?: V2FilterField[];
    detailFields?: V2FieldHint[];
  };
  uiHints?: {
    icon?: string;
    nameField?: string;
    detailSections?: string[];
  };
  actions?: string[];
};

export type PluginInfo = {
  name: string;
  version: string;
  description: string;
  displayName?: string;
  icon?: string;
};

export type PluginCapabilitiesV2 = {
  schemaVersion: string;
  plugin: PluginInfo;
  entities: EntityCapabilities[];
  sources?: {
    manageable: boolean;
    refreshable: boolean;
    types?: string[];
  };
  actions?: ActionDefinition[];
};

export type PluginsResponse = {
  plugins: PluginInfo[];
};

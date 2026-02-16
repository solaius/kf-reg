export type SourceStatus = {
  state: 'available' | 'error' | 'disabled' | 'loading';
  lastRefreshTime?: string;
  entityCount: number;
  error?: string;
};

export type SourceInfo = {
  id: string;
  name: string;
  type: string;
  enabled: boolean;
  labels?: string[];
  properties?: Record<string, unknown>;
  status: SourceStatus;
};

export type SourceConfigInput = {
  id: string;
  name: string;
  type: string;
  enabled?: boolean;
  labels?: string[];
  properties?: Record<string, unknown>;
};

export type ValidationError = {
  field?: string;
  message: string;
};

export type ValidationResult = {
  valid: boolean;
  errors?: ValidationError[];
};

export type RefreshResult = {
  sourceId?: string;
  entitiesLoaded: number;
  entitiesRemoved: number;
  duration: number;
  error?: string;
};

export type SourceDiagnostic = {
  id: string;
  name: string;
  state: string;
  entityCount: number;
  lastRefreshTime?: string;
  lastRefreshDuration?: number;
  error?: string;
};

export type DiagnosticError = {
  source?: string;
  message: string;
  time: string;
};

export type PluginDiagnostics = {
  pluginName: string;
  sources: SourceDiagnostic[];
  lastRefresh?: string;
  errors?: DiagnosticError[];
};

export type ColumnHint = {
  field: string;
  label: string;
  sortable?: boolean;
  filterable?: boolean;
};

export type FieldHint = {
  field: string;
  label: string;
  section?: string;
};

export type UIHints = {
  listColumns?: ColumnHint[];
  detailFields?: FieldHint[];
  identityField?: string;
  displayNameField?: string;
  descriptionField?: string;
};

export type ManagementCaps = {
  sourceManager: boolean;
  refresh: boolean;
  diagnostics: boolean;
};

export type CatalogPluginInfo = {
  name: string;
  version: string;
  description: string;
  basePath: string;
  healthy: boolean;
  entityKinds?: string[];
  management?: ManagementCaps;
  uiHints?: UIHints;
  cliHints?: {
    defaultColumns?: string[];
    sortField?: string;
    filterableFields?: string[];
  };
};

export type CatalogPluginList = {
  plugins: CatalogPluginInfo[];
};

export type SourcesListResponse = {
  sources: SourceInfo[];
  count: number;
};

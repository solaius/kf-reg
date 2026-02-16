import { useQueryParamNamespaces, APIState, useAPIState } from 'mod-arch-core';
import * as React from 'react';
import {
  getCatalogPlugins,
  getPluginSources,
  validatePluginSource,
  applyPluginSource,
  enablePluginSource,
  deletePluginSource,
  refreshPlugin,
  getPluginDiagnostics,
  validatePluginSourceAction,
  getPluginSourceRevisions,
  rollbackPluginSource,
  applyPluginSourceWithRefresh,
} from '~/app/api/catalogManagement/service';
import {
  ApplyResult,
  CatalogPluginInfo,
  CatalogPluginList,
  DetailedValidationResult,
  PluginDiagnostics,
  RefreshResult,
  RevisionsResponse,
  SourceConfigInput,
  SourcesListResponse,
  ValidationResult,
} from '~/app/catalogManagementTypes';
import { BFF_API_VERSION, URL_PREFIX } from '~/app/utilities/const';

export type CatalogManagementAPIs = {
  getCatalogPlugins: ReturnType<typeof getCatalogPlugins>;
  getPluginSources: ReturnType<typeof getPluginSources>;
  validatePluginSource: ReturnType<typeof validatePluginSource>;
  applyPluginSource: ReturnType<typeof applyPluginSource>;
  enablePluginSource: ReturnType<typeof enablePluginSource>;
  deletePluginSource: ReturnType<typeof deletePluginSource>;
  refreshPlugin: ReturnType<typeof refreshPlugin>;
  getPluginDiagnostics: ReturnType<typeof getPluginDiagnostics>;
  validatePluginSourceAction: ReturnType<typeof validatePluginSourceAction>;
  getPluginSourceRevisions: ReturnType<typeof getPluginSourceRevisions>;
  rollbackPluginSource: ReturnType<typeof rollbackPluginSource>;
  applyPluginSourceWithRefresh: ReturnType<typeof applyPluginSourceWithRefresh>;
};

export type CatalogManagementAPIState = APIState<CatalogManagementAPIs>;

export type CatalogManagementContextType = {
  apiState: CatalogManagementAPIState;
  refreshAPIState: () => void;
  plugins: CatalogPluginInfo[];
  pluginsLoaded: boolean;
  pluginsLoadError?: Error;
  selectedPlugin: CatalogPluginInfo | undefined;
  setSelectedPlugin: (plugin: CatalogPluginInfo | undefined) => void;
  /** When true, mutation actions (add/delete/enable/refresh) are disabled. Wire to RBAC when available. */
  isReadOnly: boolean;
};

export const CatalogManagementContext = React.createContext<CatalogManagementContextType>({
  // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
  apiState: { apiAvailable: false, api: null as unknown as CatalogManagementAPIState['api'] },
  refreshAPIState: () => undefined,
  plugins: [],
  pluginsLoaded: false,
  pluginsLoadError: undefined,
  selectedPlugin: undefined,
  setSelectedPlugin: () => undefined,
  isReadOnly: false,
});

type CatalogManagementContextProviderProps = {
  children: React.ReactNode;
};

export const CatalogManagementContextProvider: React.FC<CatalogManagementContextProviderProps> = ({
  children,
}) => {
  const hostPath = `${URL_PREFIX}/api/${BFF_API_VERSION}/model_catalog`;
  const queryParams = useQueryParamNamespaces();

  const createAPI = React.useCallback(
    (path: string) => ({
      getCatalogPlugins: getCatalogPlugins(path, queryParams),
      getPluginSources: getPluginSources(path, queryParams),
      validatePluginSource: validatePluginSource(path, queryParams),
      applyPluginSource: applyPluginSource(path, queryParams),
      enablePluginSource: enablePluginSource(path, queryParams),
      deletePluginSource: deletePluginSource(path, queryParams),
      refreshPlugin: refreshPlugin(path, queryParams),
      getPluginDiagnostics: getPluginDiagnostics(path, queryParams),
      validatePluginSourceAction: validatePluginSourceAction(path, queryParams),
      getPluginSourceRevisions: getPluginSourceRevisions(path, queryParams),
      rollbackPluginSource: rollbackPluginSource(path, queryParams),
      applyPluginSourceWithRefresh: applyPluginSourceWithRefresh(path, queryParams),
    }),
    [queryParams],
  );

  const [apiState, refreshAPIState] = useAPIState(hostPath, createAPI);

  const [plugins, setPlugins] = React.useState<CatalogPluginInfo[]>([]);
  const [pluginsLoaded, setPluginsLoaded] = React.useState(false);
  const [pluginsLoadError, setPluginsLoadError] = React.useState<Error | undefined>();
  const [selectedPlugin, setSelectedPlugin] = React.useState<CatalogPluginInfo | undefined>();

  React.useEffect(() => {
    if (!apiState.apiAvailable) {
      return;
    }

    setPluginsLoaded(false);
    apiState.api
      .getCatalogPlugins({})
      .then((data: CatalogPluginList) => {
        setPlugins(data.plugins || []);
        setPluginsLoaded(true);
        setPluginsLoadError(undefined);
      })
      .catch((err: Error) => {
        setPluginsLoadError(err);
        setPluginsLoaded(true);
      });
  }, [apiState]);

  // TODO: Wire isReadOnly to actual RBAC when available.
  // For now, defaults to false (operator role / full access).
  const isReadOnly = false;

  const contextValue = React.useMemo(
    () => ({
      apiState,
      refreshAPIState,
      plugins,
      pluginsLoaded,
      pluginsLoadError,
      selectedPlugin,
      setSelectedPlugin,
      isReadOnly,
    }),
    [apiState, refreshAPIState, plugins, pluginsLoaded, pluginsLoadError, selectedPlugin, isReadOnly],
  );

  return (
    <CatalogManagementContext.Provider value={contextValue}>
      {children}
    </CatalogManagementContext.Provider>
  );
};

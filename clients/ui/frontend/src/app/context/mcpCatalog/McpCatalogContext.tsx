import { useQueryParamNamespaces, APIState, useAPIState } from 'mod-arch-core';
import useGenericObjectState from 'mod-arch-core/dist/utilities/useGenericObjectState';
import * as React from 'react';
import { getMcpServers, getMcpServer } from '~/app/api/mcpCatalog/service';
import {
  McpServer,
  McpServerList,
  McpCatalogFilterKey,
  McpCatalogFilterStates,
} from '~/app/mcpCatalogTypes';
import { BFF_API_VERSION, URL_PREFIX } from '~/app/utilities/const';

export type McpCatalogAPIs = {
  getMcpServers: ReturnType<typeof getMcpServers>;
  getMcpServer: ReturnType<typeof getMcpServer>;
};

export type McpCatalogAPIState = APIState<McpCatalogAPIs>;

export type McpCatalogContextType = {
  apiState: McpCatalogAPIState;
  mcpServers: McpServer[];
  mcpServersLoaded: boolean;
  mcpServersLoadError?: Error;
  searchTerm: string;
  setSearchTerm: (term: string) => void;
  filterData: McpCatalogFilterStates;
  setFilterData: <K extends keyof McpCatalogFilterStates>(
    key: K,
    value: McpCatalogFilterStates[K],
  ) => void;
  filteredServers: McpServer[];
};

export const McpCatalogContext = React.createContext<McpCatalogContextType>({
  // eslint-disable-next-line @typescript-eslint/consistent-type-assertions
  apiState: { apiAvailable: false, api: null as unknown as McpCatalogAPIState['api'] },
  mcpServers: [],
  mcpServersLoaded: false,
  mcpServersLoadError: undefined,
  searchTerm: '',
  setSearchTerm: () => undefined,
  filterData: {
    [McpCatalogFilterKey.DEPLOYMENT_MODE]: [],
    [McpCatalogFilterKey.CATEGORY]: [],
    [McpCatalogFilterKey.LICENSE]: [],
    [McpCatalogFilterKey.TRANSPORT]: [],
  },
  setFilterData: () => undefined,
  filteredServers: [],
});

type McpCatalogContextProviderProps = {
  children: React.ReactNode;
};

export const McpCatalogContextProvider: React.FC<McpCatalogContextProviderProps> = ({
  children,
}) => {
  const hostPath = `${URL_PREFIX}/api/${BFF_API_VERSION}/model_catalog`;
  const queryParams = useQueryParamNamespaces();

  const createAPI = React.useCallback(
    (path: string) => ({
      getMcpServers: getMcpServers(path, queryParams),
      getMcpServer: getMcpServer(path, queryParams),
    }),
    [queryParams],
  );

  const [apiState] = useAPIState(hostPath, createAPI);

  const [mcpServers, setMcpServers] = React.useState<McpServer[]>([]);
  const [mcpServersLoaded, setMcpServersLoaded] = React.useState(false);
  const [mcpServersLoadError, setMcpServersLoadError] = React.useState<Error | undefined>();
  const [searchTerm, setSearchTerm] = React.useState('');
  const [filterData, setFilterData] = useGenericObjectState<McpCatalogFilterStates>({
    [McpCatalogFilterKey.DEPLOYMENT_MODE]: [],
    [McpCatalogFilterKey.CATEGORY]: [],
    [McpCatalogFilterKey.LICENSE]: [],
    [McpCatalogFilterKey.TRANSPORT]: [],
  });

  React.useEffect(() => {
    if (!apiState.apiAvailable) {
      return;
    }

    setMcpServersLoaded(false);
    apiState.api
      .getMcpServers({})
      .then((data: McpServerList) => {
        setMcpServers(data.items || []);
        setMcpServersLoaded(true);
        setMcpServersLoadError(undefined);
      })
      .catch((err: Error) => {
        setMcpServersLoadError(err);
        setMcpServersLoaded(true);
      });
  }, [apiState]);

  // Client-side filtering
  const filteredServers = React.useMemo(() => {
    let result = mcpServers;

    // Search filter
    if (searchTerm) {
      const term = searchTerm.toLowerCase();
      result = result.filter(
        (s) =>
          s.name.toLowerCase().includes(term) ||
          (s.description && s.description.toLowerCase().includes(term)) ||
          (s.provider && s.provider.toLowerCase().includes(term)),
      );
    }

    // Deployment mode filter
    if (filterData[McpCatalogFilterKey.DEPLOYMENT_MODE].length > 0) {
      result = result.filter((s) =>
        filterData[McpCatalogFilterKey.DEPLOYMENT_MODE].includes(s.deploymentMode || ''),
      );
    }

    // Category filter
    if (filterData[McpCatalogFilterKey.CATEGORY].length > 0) {
      result = result.filter((s) =>
        filterData[McpCatalogFilterKey.CATEGORY].includes(s.category || ''),
      );
    }

    // License filter
    if (filterData[McpCatalogFilterKey.LICENSE].length > 0) {
      result = result.filter((s) =>
        filterData[McpCatalogFilterKey.LICENSE].includes(s.license || ''),
      );
    }

    // Transport filter
    if (filterData[McpCatalogFilterKey.TRANSPORT].length > 0) {
      result = result.filter((s) =>
        filterData[McpCatalogFilterKey.TRANSPORT].some(
          (t) =>
            s.transportType === t ||
            (s.supportedTransports && s.supportedTransports.split(',').includes(t)),
        ),
      );
    }

    return result;
  }, [mcpServers, searchTerm, filterData]);

  const contextValue = React.useMemo(
    () => ({
      apiState,
      mcpServers,
      mcpServersLoaded,
      mcpServersLoadError,
      searchTerm,
      setSearchTerm,
      filterData,
      setFilterData,
      filteredServers,
    }),
    [
      apiState,
      mcpServers,
      mcpServersLoaded,
      mcpServersLoadError,
      searchTerm,
      filterData,
      filteredServers,
      setFilterData,
    ],
  );

  return (
    <McpCatalogContext.Provider value={contextValue}>{children}</McpCatalogContext.Provider>
  );
};

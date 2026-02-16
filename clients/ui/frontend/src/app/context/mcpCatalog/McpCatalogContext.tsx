import { useQueryParamNamespaces, APIState, useAPIState } from 'mod-arch-core';
import useGenericObjectState from 'mod-arch-core/dist/utilities/useGenericObjectState';
import * as React from 'react';
import { getMcpServers, getMcpServer } from '~/app/api/mcpCatalog/service';
import {
  McpServer,
  McpServerList,
  McpCatalogFilterKey,
  McpCatalogFilterStates,
  MCP_ALL_CATEGORIES,
  MCP_OTHER_SERVERS,
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
  selectedCategory: string;
  setSelectedCategory: (category: string) => void;
  clearAllFilters: () => void;
  availableFilterValues: {
    deploymentModes: string[];
    categories: string[];
    licenses: string[];
    transports: string[];
    sourceLabels: string[];
  };
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
  selectedCategory: MCP_ALL_CATEGORIES,
  setSelectedCategory: () => undefined,
  clearAllFilters: () => undefined,
  availableFilterValues: {
    deploymentModes: [],
    categories: [],
    licenses: [],
    transports: [],
    sourceLabels: [],
  },
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
  const [selectedCategory, setSelectedCategory] = React.useState(MCP_ALL_CATEGORIES);
  const [filterData, setFilterData] = useGenericObjectState<McpCatalogFilterStates>({
    [McpCatalogFilterKey.DEPLOYMENT_MODE]: [],
    [McpCatalogFilterKey.CATEGORY]: [],
    [McpCatalogFilterKey.LICENSE]: [],
    [McpCatalogFilterKey.TRANSPORT]: [],
  });

  const availableFilterValues = React.useMemo(() => {
    const deploymentModes = [
      ...new Set(mcpServers.map((s) => s.deploymentMode).filter(Boolean)),
    ] as string[];
    const categories = [
      ...new Set(mcpServers.map((s) => s.category).filter(Boolean)),
    ] as string[];
    const licenses = [
      ...new Set(mcpServers.map((s) => s.license).filter(Boolean)),
    ] as string[];
    const transports = [
      ...new Set(
        mcpServers.flatMap((s) => {
          const t = s.supportedTransports || s.transportType || '';
          return t
            .split(',')
            .map((v) => v.trim())
            .filter(Boolean);
        }),
      ),
    ];
    const sourceLabels = [
      ...new Set(mcpServers.map((s) => s.sourceLabel).filter(Boolean)),
    ] as string[];
    return { deploymentModes, categories, licenses, transports, sourceLabels };
  }, [mcpServers]);

  const clearAllFilters = React.useCallback(() => {
    setFilterData(McpCatalogFilterKey.DEPLOYMENT_MODE, []);
    setFilterData(McpCatalogFilterKey.CATEGORY, []);
    setFilterData(McpCatalogFilterKey.LICENSE, []);
    setFilterData(McpCatalogFilterKey.TRANSPORT, []);
    setSelectedCategory(MCP_ALL_CATEGORIES);
    setSearchTerm('');
  }, [setFilterData]);

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

    // Source label tab filter
    if (selectedCategory !== MCP_ALL_CATEGORIES) {
      if (selectedCategory === MCP_OTHER_SERVERS) {
        result = result.filter((s) => !s.sourceLabel);
      } else {
        result = result.filter((s) => s.sourceLabel === selectedCategory);
      }
    }

    return result;
  }, [mcpServers, searchTerm, filterData, selectedCategory]);

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
      selectedCategory,
      setSelectedCategory,
      clearAllFilters,
      availableFilterValues,
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
      selectedCategory,
      clearAllFilters,
      availableFilterValues,
    ],
  );

  return (
    <McpCatalogContext.Provider value={contextValue}>{children}</McpCatalogContext.Provider>
  );
};

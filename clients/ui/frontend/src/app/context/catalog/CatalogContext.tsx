import * as React from 'react';
import { useAPIState, APIState } from 'mod-arch-core';
import { getAllPlugins, getPluginCapabilities } from '~/app/api/catalogCapabilities/service';
import {
  PluginInfo,
  PluginCapabilitiesV2,
  PluginsResponse,
} from '~/app/types/capabilities';
import { BFF_API_VERSION, URL_PREFIX } from '~/app/utilities/const';

type CatalogAPIs = {
  getAllPlugins: ReturnType<typeof getAllPlugins>;
};

type CatalogAPIState = APIState<CatalogAPIs>;

export type CatalogContextType = {
  plugins: PluginInfo[];
  pluginsLoaded: boolean;
  pluginsLoadError?: Error;
  capabilitiesMap: Record<string, PluginCapabilitiesV2>;
  getPluginCaps: (pluginName: string) => PluginCapabilitiesV2 | undefined;
};

export const CatalogContext = React.createContext<CatalogContextType>({
  plugins: [],
  pluginsLoaded: false,
  pluginsLoadError: undefined,
  capabilitiesMap: {},
  getPluginCaps: () => undefined,
});

type CatalogContextProviderProps = {
  children: React.ReactNode;
};

export const CatalogContextProvider: React.FC<CatalogContextProviderProps> = ({ children }) => {
  const hostPath = `${URL_PREFIX}/api/${BFF_API_VERSION}/model_catalog`;

  const createAPI = React.useCallback(
    (path: string) => ({
      getAllPlugins: getAllPlugins(path),
    }),
    [],
  );

  const [apiState] = useAPIState<CatalogAPIs>(hostPath, createAPI);

  const [plugins, setPlugins] = React.useState<PluginInfo[]>([]);
  const [pluginsLoaded, setPluginsLoaded] = React.useState(false);
  const [pluginsLoadError, setPluginsLoadError] = React.useState<Error | undefined>();
  const [capabilitiesMap, setCapabilitiesMap] = React.useState<
    Record<string, PluginCapabilitiesV2>
  >({});

  // Fetch plugins list
  React.useEffect(() => {
    if (!apiState.apiAvailable) {
      return;
    }

    setPluginsLoaded(false);
    apiState.api
      .getAllPlugins({})
      .then((data: PluginsResponse) => {
        const pluginList = data.plugins || [];
        setPlugins(pluginList);
        setPluginsLoaded(true);
        setPluginsLoadError(undefined);

        // Fetch capabilities for each plugin
        pluginList.forEach((plugin) => {
          const fetchCaps = getPluginCapabilities(hostPath, plugin.name);
          fetchCaps({})
            .then((caps: PluginCapabilitiesV2) => {
              setCapabilitiesMap((prev) => ({ ...prev, [plugin.name]: caps }));
            })
            .catch(() => {
              // Silently skip plugins whose capabilities fail to load
            });
        });
      })
      .catch((err: Error) => {
        setPluginsLoadError(err);
        setPluginsLoaded(true);
      });
  }, [apiState, hostPath]);

  const getPluginCaps = React.useCallback(
    (pluginName: string): PluginCapabilitiesV2 | undefined => capabilitiesMap[pluginName],
    [capabilitiesMap],
  );

  const contextValue = React.useMemo(
    () => ({
      plugins,
      pluginsLoaded,
      pluginsLoadError,
      capabilitiesMap,
      getPluginCaps,
    }),
    [plugins, pluginsLoaded, pluginsLoadError, capabilitiesMap, getPluginCaps],
  );

  return <CatalogContext.Provider value={contextValue}>{children}</CatalogContext.Provider>;
};

export const useCatalogPlugins = (): CatalogContextType => React.useContext(CatalogContext);

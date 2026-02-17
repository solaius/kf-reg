import { APIOptions, handleRestFailures, isModArchResponse, restGET } from 'mod-arch-core';
import { PluginCapabilitiesV2, PluginsResponse } from '~/app/types/capabilities';

export const getAllPlugins =
  (hostPath: string) =>
  (opts: APIOptions): Promise<PluginsResponse> =>
    handleRestFailures(restGET(hostPath, '/../model_catalog/plugins', {}, opts)).then((response) => {
      if (isModArchResponse<PluginsResponse>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const getPluginCapabilities =
  (hostPath: string, pluginName: string) =>
  (opts: APIOptions): Promise<PluginCapabilitiesV2> =>
    handleRestFailures(
      restGET(hostPath, `/../catalog/${pluginName}/capabilities`, {}, opts),
    ).then((response) => {
      if (isModArchResponse<PluginCapabilitiesV2>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

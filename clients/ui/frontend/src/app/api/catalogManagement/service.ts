import {
  APIOptions,
  assembleModArchBody,
  handleRestFailures,
  isModArchResponse,
  restCREATE,
  restDELETE,
  restGET,
} from 'mod-arch-core';
import {
  CatalogPluginList,
  PluginDiagnostics,
  RefreshResult,
  SourceConfigInput,
  SourcesListResponse,
  ValidationResult,
} from '~/app/catalogManagementTypes';

export const getCatalogPlugins =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions): Promise<CatalogPluginList> =>
    handleRestFailures(restGET(hostPath, '/plugins', queryParams, opts)).then((response) => {
      if (isModArchResponse<CatalogPluginList>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const getPluginSources =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, pluginName: string): Promise<SourcesListResponse> =>
    handleRestFailures(
      restGET(hostPath, `/../catalog/${pluginName}/sources`, queryParams, opts),
    ).then((response) => {
      if (isModArchResponse<SourcesListResponse>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const validatePluginSource =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, pluginName: string, data: SourceConfigInput): Promise<ValidationResult> =>
    handleRestFailures(
      restCREATE(
        hostPath,
        `/../catalog/${pluginName}/validate-source`,
        assembleModArchBody(data),
        queryParams,
        opts,
      ),
    ).then((response) => {
      if (isModArchResponse<ValidationResult>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const applyPluginSource =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, pluginName: string, data: SourceConfigInput): Promise<void> =>
    handleRestFailures(
      restCREATE(
        hostPath,
        `/../catalog/${pluginName}/apply-source`,
        assembleModArchBody(data),
        queryParams,
        opts,
      ),
    ).then(() => undefined);

export const enablePluginSource =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, pluginName: string, sourceId: string, enabled: boolean): Promise<void> =>
    handleRestFailures(
      restCREATE(
        hostPath,
        `/../catalog/${pluginName}/sources/${sourceId}/enable`,
        assembleModArchBody({ enabled }),
        queryParams,
        opts,
      ),
    ).then(() => undefined);

export const deletePluginSource =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, pluginName: string, sourceId: string): Promise<void> =>
    handleRestFailures(
      restDELETE(hostPath, `/../catalog/${pluginName}/sources/${sourceId}`, {}, queryParams, opts),
    );

export const refreshPlugin =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, pluginName: string, sourceId?: string): Promise<RefreshResult> =>
    handleRestFailures(
      restCREATE(
        hostPath,
        `/../catalog/${pluginName}/refresh${sourceId ? `/${sourceId}` : ''}`,
        assembleModArchBody({}),
        queryParams,
        opts,
      ),
    ).then((response) => {
      if (isModArchResponse<RefreshResult>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const getPluginDiagnostics =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, pluginName: string): Promise<PluginDiagnostics> =>
    handleRestFailures(
      restGET(hostPath, `/../catalog/${pluginName}/diagnostics`, queryParams, opts),
    ).then((response) => {
      if (isModArchResponse<PluginDiagnostics>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

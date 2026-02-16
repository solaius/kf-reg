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
  ApplyResult,
  CatalogPluginList,
  DetailedValidationResult,
  PluginDiagnostics,
  RefreshResult,
  RevisionsResponse,
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

export const validatePluginSourceAction =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (
    opts: APIOptions,
    pluginName: string,
    sourceId: string,
    data: SourceConfigInput,
  ): Promise<DetailedValidationResult> =>
    handleRestFailures(
      restCREATE(
        hostPath,
        `/../catalog/${pluginName}/sources/${sourceId}/validate`,
        assembleModArchBody(data),
        queryParams,
        opts,
      ),
    ).then((response) => {
      if (isModArchResponse<DetailedValidationResult>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const getPluginSourceRevisions =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, pluginName: string, sourceId: string): Promise<RevisionsResponse> =>
    handleRestFailures(
      restGET(hostPath, `/../catalog/${pluginName}/sources/${sourceId}/revisions`, queryParams, opts),
    ).then((response) => {
      if (isModArchResponse<RevisionsResponse>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const rollbackPluginSource =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, pluginName: string, sourceId: string, version: string): Promise<void> =>
    handleRestFailures(
      restCREATE(
        hostPath,
        `/../catalog/${pluginName}/sources/${sourceId}/rollback`,
        assembleModArchBody({ version }),
        queryParams,
        opts,
      ),
    ).then(() => undefined);

export const applyPluginSourceWithRefresh =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, pluginName: string, data: SourceConfigInput & { refreshAfterApply?: boolean }): Promise<ApplyResult> =>
    handleRestFailures(
      restCREATE(
        hostPath,
        `/../catalog/${pluginName}/apply-source`,
        assembleModArchBody(data),
        queryParams,
        opts,
      ),
    ).then((response) => {
      if (isModArchResponse<ApplyResult>(response)) {
        return response.data;
      }
      // Fall back to a default result when the response doesn't include ApplyResult data
      return { status: 'applied' };
    });

import {
  APIOptions,
  handleRestFailures,
  isModArchResponse,
  restGET,
  restCREATE,
} from 'mod-arch-core';
import { GenericEntity, GenericEntityList } from '~/app/types/asset';

export const getEntityList =
  (
    hostPath: string,
    pluginName: string,
    entityPlural: string,
    queryParams: Record<string, unknown> = {},
  ) =>
  (opts: APIOptions): Promise<GenericEntityList> =>
    handleRestFailures(
      restGET(
        hostPath,
        `/../catalog/${pluginName}/entities/${entityPlural}`,
        queryParams,
        opts,
      ),
    ).then((response) => {
      if (isModArchResponse<GenericEntityList>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const getEntity =
  (hostPath: string, pluginName: string, entityPlural: string, entityName: string) =>
  (opts: APIOptions): Promise<GenericEntity> =>
    handleRestFailures(
      restGET(
        hostPath,
        `/../catalog/${pluginName}/entities/${entityPlural}/${entityName}`,
        {},
        opts,
      ),
    ).then((response) => {
      if (isModArchResponse<GenericEntity>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const executeEntityAction =
  (
    hostPath: string,
    pluginName: string,
    entityPlural: string,
    entityName: string,
    actionId: string,
    params: Record<string, unknown> = {},
  ) =>
  (opts: APIOptions): Promise<GenericEntity> =>
    handleRestFailures(
      restCREATE(
        hostPath,
        `/../catalog/${pluginName}/entities/${entityPlural}/${entityName}/action`,
        { action: actionId, ...params },
        {},
        opts,
      ),
    ).then((response) => {
      if (isModArchResponse<GenericEntity>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

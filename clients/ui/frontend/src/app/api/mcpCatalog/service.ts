import {
  APIOptions,
  handleRestFailures,
  isModArchResponse,
  restGET,
} from 'mod-arch-core';
import { McpServer, McpServerList } from '~/app/mcpCatalogTypes';

export const getMcpServers =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions): Promise<McpServerList> =>
    handleRestFailures(restGET(hostPath, '/../mcp_catalog/mcpservers', queryParams, opts)).then(
      (response) => {
        if (isModArchResponse<McpServerList>(response)) {
          return response.data;
        }
        throw new Error('Invalid response format');
      },
    );

export const getMcpServer =
  (hostPath: string, queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions, name: string): Promise<McpServer> =>
    handleRestFailures(
      restGET(hostPath, `/../mcp_catalog/mcpservers/${name}`, queryParams, opts),
    ).then((response) => {
      if (isModArchResponse<McpServer>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

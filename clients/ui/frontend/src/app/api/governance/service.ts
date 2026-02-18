import {
  APIOptions,
  handleRestFailures,
  isModArchResponse,
  restGET,
  restCREATE,
  restPATCH,
} from 'mod-arch-core';
import {
  GovernanceResponse,
  AuditEventList,
  ActionResult,
  VersionListResponse,
  VersionResponse,
  BindingsResponse,
  ApprovalRequestList,
  ApprovalRequest,
  GovernanceOverlay,
} from '~/app/types/governance';
import { BFF_API_VERSION, URL_PREFIX } from '~/app/utilities/const';

const governancePath = `${URL_PREFIX}/api/${BFF_API_VERSION}/governance`;

export const getGovernance =
  (plugin: string, kind: string, name: string) =>
  (opts: APIOptions): Promise<GovernanceResponse> =>
    handleRestFailures(
      restGET(governancePath, `/assets/${plugin}/${kind}/${name}`, {}, opts),
    ).then((response) => {
      if (isModArchResponse<GovernanceResponse>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const patchGovernance =
  (plugin: string, kind: string, name: string, overlay: Partial<GovernanceOverlay>) =>
  (opts: APIOptions): Promise<GovernanceResponse> =>
    handleRestFailures(
      restPATCH(governancePath, `/assets/${plugin}/${kind}/${name}`, overlay, {}, opts),
    ).then((response) => {
      if (isModArchResponse<GovernanceResponse>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const getGovernanceHistory =
  (
    plugin: string,
    kind: string,
    name: string,
    queryParams: Record<string, unknown> = {},
  ) =>
  (opts: APIOptions): Promise<AuditEventList> =>
    handleRestFailures(
      restGET(governancePath, `/assets/${plugin}/${kind}/${name}/history`, queryParams, opts),
    ).then((response) => {
      if (isModArchResponse<AuditEventList>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const postGovernanceAction =
  (
    plugin: string,
    kind: string,
    name: string,
    action: string,
    params: Record<string, unknown> = {},
    dryRun = false,
  ) =>
  (opts: APIOptions): Promise<ActionResult> =>
    handleRestFailures(
      restCREATE(
        governancePath,
        `/assets/${plugin}/${kind}/${name}/actions/${action}`,
        { dryRun, params },
        {},
        opts,
      ),
    ).then((response) => {
      if (isModArchResponse<ActionResult>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const listVersions =
  (
    plugin: string,
    kind: string,
    name: string,
    queryParams: Record<string, unknown> = {},
  ) =>
  (opts: APIOptions): Promise<VersionListResponse> =>
    handleRestFailures(
      restGET(governancePath, `/assets/${plugin}/${kind}/${name}/versions`, queryParams, opts),
    ).then((response) => {
      if (isModArchResponse<VersionListResponse>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const createVersion =
  (plugin: string, kind: string, name: string, versionLabel: string, reason = '') =>
  (opts: APIOptions): Promise<VersionResponse> =>
    handleRestFailures(
      restCREATE(
        governancePath,
        `/assets/${plugin}/${kind}/${name}/versions`,
        { versionLabel, reason },
        {},
        opts,
      ),
    ).then((response) => {
      if (isModArchResponse<VersionResponse>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const listBindings =
  (plugin: string, kind: string, name: string) =>
  (opts: APIOptions): Promise<BindingsResponse> =>
    handleRestFailures(
      restGET(governancePath, `/assets/${plugin}/${kind}/${name}/bindings`, {}, opts),
    ).then((response) => {
      if (isModArchResponse<BindingsResponse>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const setBinding =
  (plugin: string, kind: string, name: string, env: string, versionId: string) =>
  (opts: APIOptions): Promise<unknown> =>
    handleRestFailures(
      restPATCH(
        governancePath,
        `/assets/${plugin}/${kind}/${name}/bindings/${env}`,
        { versionId },
        {},
        opts,
      ),
    ).then((response) => {
      if (isModArchResponse<unknown>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const listApprovals =
  (queryParams: Record<string, unknown> = {}) =>
  (opts: APIOptions): Promise<ApprovalRequestList> =>
    handleRestFailures(
      restGET(governancePath, `/approvals`, queryParams, opts),
    ).then((response) => {
      if (isModArchResponse<ApprovalRequestList>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const getApproval =
  (id: string) =>
  (opts: APIOptions): Promise<ApprovalRequest> =>
    handleRestFailures(
      restGET(governancePath, `/approvals/${id}`, {}, opts),
    ).then((response) => {
      if (isModArchResponse<ApprovalRequest>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const submitDecision =
  (id: string, verdict: 'approve' | 'deny', comment = '') =>
  (opts: APIOptions): Promise<unknown> =>
    handleRestFailures(
      restCREATE(governancePath, `/approvals/${id}/decisions`, { verdict, comment }, {}, opts),
    ).then((response) => {
      if (isModArchResponse<unknown>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const cancelApproval =
  (id: string, reason = '') =>
  (opts: APIOptions): Promise<unknown> =>
    handleRestFailures(
      restCREATE(governancePath, `/approvals/${id}/cancel`, { reason }, {}, opts),
    ).then((response) => {
      if (isModArchResponse<unknown>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

export const listPolicies =
  () =>
  (opts: APIOptions): Promise<unknown> =>
    handleRestFailures(
      restGET(governancePath, `/policies`, {}, opts),
    ).then((response) => {
      if (isModArchResponse<unknown>(response)) {
        return response.data;
      }
      throw new Error('Invalid response format');
    });

import * as React from 'react';
import {
  Alert,
  Button,
  Label,
  Spinner,
  Switch,
  Toolbar,
  ToolbarContent,
  ToolbarItem,
  Tooltip,
} from '@patternfly/react-core';
import { PlusCircleIcon, SyncIcon } from '@patternfly/react-icons';
import {
  ActionsColumn,
  IAction,
  Table,
  Thead,
  Tr,
  Th,
  Tbody,
  Td,
} from '@patternfly/react-table';
import { useNavigate } from 'react-router-dom';
import { CatalogManagementContext } from '~/app/context/catalogManagement/CatalogManagementContext';
import { SourceInfo } from '~/app/catalogManagementTypes';
import {
  catalogPluginAddSourceUrl,
  catalogPluginManageSourceUrl,
} from '~/app/routes/catalogManagement/catalogManagement';
import {
  addSourceUrl as modelCatalogAddSourceUrl,
  manageSourceUrl as modelCatalogManageSourceUrl,
} from '~/app/routes/modelCatalogSettings/modelCatalogSettings';
import DeleteModal from '~/app/shared/components/DeleteModal';

const statusLabelColor = (
  state: string,
): 'green' | 'red' | 'grey' | 'blue' => {
  switch (state) {
    case 'available':
      return 'green';
    case 'error':
      return 'red';
    case 'disabled':
      return 'grey';
    case 'loading':
      return 'blue';
    default:
      return 'grey';
  }
};

const sourceTypeLabel = (type: string): string => {
  switch (type) {
    case 'yaml':
      return 'YAML file';
    case 'huggingface':
      return 'Hugging Face';
    default:
      return type;
  }
};

const READONLY_TOOLTIP = 'Operator role required';

const PluginSourcesPage: React.FC = () => {
  const navigate = useNavigate();
  const { apiState, selectedPlugin, isReadOnly } = React.useContext(CatalogManagementContext);

  const [sources, setSources] = React.useState<SourceInfo[]>([]);
  const [loaded, setLoaded] = React.useState(false);
  const [loadError, setLoadError] = React.useState<string | undefined>();
  const [refreshing, setRefreshing] = React.useState(false);
  const [refreshMessage, setRefreshMessage] = React.useState<string | undefined>();
  const [deleteSource, setDeleteSource] = React.useState<SourceInfo | undefined>();
  const [deleting, setDeleting] = React.useState(false);
  const [deleteError, setDeleteError] = React.useState<Error | undefined>();

  const fetchSources = React.useCallback(() => {
    if (!apiState.apiAvailable || !selectedPlugin) {
      return;
    }
    setLoaded(false);
    apiState.api
      .getPluginSources({}, selectedPlugin.name)
      .then((data) => {
        setSources(data.sources || []);
        setLoaded(true);
        setLoadError(undefined);
      })
      .catch((err: Error) => {
        setLoadError(err.message);
        setLoaded(true);
      });
  }, [apiState, selectedPlugin]);

  React.useEffect(() => {
    fetchSources();
  }, [fetchSources]);

  const handleRefreshAll = React.useCallback(async () => {
    if (!apiState.apiAvailable || !selectedPlugin) {
      return;
    }
    setRefreshing(true);
    setRefreshMessage(undefined);
    try {
      const result = await apiState.api.refreshPlugin({}, selectedPlugin.name);
      if (result.error) {
        setRefreshMessage(`Refresh error: ${result.error}`);
      } else {
        setRefreshMessage('Refresh completed successfully.');
      }
      fetchSources();
    } catch (err) {
      setRefreshMessage(`Refresh failed: ${err instanceof Error ? err.message : String(err)}`);
    } finally {
      setRefreshing(false);
    }
  }, [apiState, selectedPlugin, fetchSources]);

  const handleToggleEnable = React.useCallback(
    async (source: SourceInfo, enabled: boolean) => {
      if (!apiState.apiAvailable || !selectedPlugin) {
        return;
      }
      try {
        await apiState.api.enablePluginSource({}, selectedPlugin.name, source.id, enabled);
        fetchSources();
      } catch (err) {
        setLoadError(`Failed to toggle source: ${err instanceof Error ? err.message : String(err)}`);
      }
    },
    [apiState, selectedPlugin, fetchSources],
  );

  const handleDeleteConfirm = React.useCallback(async () => {
    if (!apiState.apiAvailable || !selectedPlugin || !deleteSource) {
      return;
    }
    setDeleting(true);
    setDeleteError(undefined);
    try {
      await apiState.api.deletePluginSource({}, selectedPlugin.name, deleteSource.id);
      setDeleteSource(undefined);
      fetchSources();
    } catch (err) {
      setDeleteError(err instanceof Error ? err : new Error(String(err)));
    } finally {
      setDeleting(false);
    }
  }, [apiState, selectedPlugin, deleteSource, fetchSources]);

  if (!selectedPlugin) {
    return null;
  }

  const hasRefresh = selectedPlugin.management?.refresh;
  const hasSourceManager = selectedPlugin.management?.sourceManager;
  const isModelPlugin = selectedPlugin.name === 'model';

  const getAddSourceUrl = (): string =>
    isModelPlugin ? modelCatalogAddSourceUrl() : catalogPluginAddSourceUrl(selectedPlugin.name);

  const getManageSourceUrl = (sourceId: string): string =>
    isModelPlugin
      ? modelCatalogManageSourceUrl(sourceId)
      : catalogPluginManageSourceUrl(selectedPlugin.name, sourceId);

  const getRowActions = (source: SourceInfo): IAction[] => [
    {
      title: 'Manage source',
      isDisabled: isReadOnly,
      tooltipProps: isReadOnly ? { content: READONLY_TOOLTIP } : undefined,
      onClick: () => {
        navigate(getManageSourceUrl(source.id));
      },
    },
    {
      title: 'View in catalog',
      onClick: () => {
        // Map plugin names to their catalog routes
        const catalogRoutes: Record<string, string> = {
          model: '/model-catalog',
          mcp: '/mcp-catalog',
        };
        const basePath = catalogRoutes[selectedPlugin.name] || `/${selectedPlugin.name}-catalog`;
        navigate(`${basePath}?source=${encodeURIComponent(source.id)}`);
      },
    },
    { isSeparator: true },
    {
      title: 'Delete source',
      isDisabled: isReadOnly,
      tooltipProps: isReadOnly ? { content: READONLY_TOOLTIP } : undefined,
      onClick: () => setDeleteSource(source),
    },
  ];

  return (
    <>
      <Toolbar>
        <ToolbarContent>
          {hasSourceManager && (
            <ToolbarItem>
              <Tooltip content={READONLY_TOOLTIP} trigger={isReadOnly ? 'mouseenter focus' : 'manual'}>
                <Button
                  variant="primary"
                  icon={<PlusCircleIcon />}
                  onClick={() => navigate(getAddSourceUrl())}
                  isDisabled={isReadOnly}
                  data-testid="add-source-button"
                >
                  Add a source
                </Button>
              </Tooltip>
            </ToolbarItem>
          )}
          {hasRefresh && (
            <ToolbarItem>
              <Tooltip content={READONLY_TOOLTIP} trigger={isReadOnly ? 'mouseenter focus' : 'manual'}>
                <Button
                  variant="secondary"
                  icon={<SyncIcon />}
                  onClick={handleRefreshAll}
                  isDisabled={refreshing || isReadOnly}
                  isLoading={refreshing}
                  data-testid="refresh-all-button"
                >
                  Refresh all sources
                </Button>
              </Tooltip>
            </ToolbarItem>
          )}
        </ToolbarContent>
      </Toolbar>

      {refreshMessage && (
        <Alert
          variant={refreshMessage.includes('error') || refreshMessage.includes('failed') ? 'danger' : 'success'}
          isInline
          title={refreshMessage}
          data-testid="refresh-alert"
        />
      )}

      {loadError && (
        <Alert variant="danger" isInline title={loadError} data-testid="load-error-alert" />
      )}

      {!loaded ? (
        <Spinner size="lg" />
      ) : (
        <Table aria-label="Plugin sources" data-testid="plugin-sources-table">
          <Thead>
            <Tr>
              <Th>Source name</Th>
              <Th>Source type</Th>
              <Th>Visible in catalog</Th>
              <Th>Status</Th>
              <Th>Entities</Th>
              {hasSourceManager && <Th>Manage source</Th>}
              {hasSourceManager && <Th />}
            </Tr>
          </Thead>
          <Tbody>
            {sources.length === 0 ? (
              <Tr>
                <Td colSpan={hasSourceManager ? 7 : 5}>No sources configured.</Td>
              </Tr>
            ) : (
              sources.map((source) => (
                <Tr key={source.id} data-testid={`source-row-${source.id}`}>
                  <Td dataLabel="Source name">{source.name}</Td>
                  <Td dataLabel="Source type">{sourceTypeLabel(source.type)}</Td>
                  <Td dataLabel="Visible in catalog">
                    <Tooltip content={READONLY_TOOLTIP} trigger={isReadOnly ? 'mouseenter focus' : 'manual'}>
                      <Switch
                        id={`toggle-${source.id}`}
                        aria-label={`Toggle visibility for ${source.name}`}
                        isChecked={source.enabled}
                        isDisabled={isReadOnly}
                        onChange={(_event, checked) => handleToggleEnable(source, checked)}
                        data-testid={`toggle-enable-${source.id}`}
                      />
                    </Tooltip>
                  </Td>
                  <Td dataLabel="Status">
                    <Label color={statusLabelColor(source.status.state)}>
                      {source.status.state}
                    </Label>
                    {source.status.error && (
                      <span title={source.status.error}> ({source.status.error})</span>
                    )}
                  </Td>
                  <Td dataLabel="Entities">{source.status.entityCount}</Td>
                  {hasSourceManager && (
                    <Td dataLabel="Manage source">
                      <Button
                        variant="link"
                        onClick={() =>
                          navigate(getManageSourceUrl(source.id))
                        }
                        data-testid={`manage-source-${source.id}`}
                      >
                        Manage source
                      </Button>
                    </Td>
                  )}
                  {hasSourceManager && (
                    <Td isActionCell>
                      <ActionsColumn items={getRowActions(source)} />
                    </Td>
                  )}
                </Tr>
              ))
            )}
          </Tbody>
        </Table>
      )}

      {deleteSource && (
        <DeleteModal
          title={`Delete source ${deleteSource.name}?`}
          onClose={() => {
            setDeleteSource(undefined);
            setDeleteError(undefined);
          }}
          deleting={deleting}
          onDelete={handleDeleteConfirm}
          deleteName={deleteSource.name}
          error={deleteError}
          testId="delete-source-modal"
        >
          <p>
            This will permanently remove the source <strong>{deleteSource.name}</strong> and all its
            entities from the catalog.
          </p>
        </DeleteModal>
      )}
    </>
  );
};

export default PluginSourcesPage;

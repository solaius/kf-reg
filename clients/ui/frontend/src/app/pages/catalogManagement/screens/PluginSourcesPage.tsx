import * as React from 'react';
import {
  Alert,
  Button,
  Label,
  Spinner,
  Toolbar,
  ToolbarContent,
  ToolbarItem,
} from '@patternfly/react-core';
import { SyncIcon } from '@patternfly/react-icons';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { CatalogManagementContext } from '~/app/context/catalogManagement/CatalogManagementContext';
import { SourceInfo } from '~/app/catalogManagementTypes';

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

const PluginSourcesPage: React.FC = () => {
  const { apiState, selectedPlugin } = React.useContext(CatalogManagementContext);

  const [sources, setSources] = React.useState<SourceInfo[]>([]);
  const [loaded, setLoaded] = React.useState(false);
  const [loadError, setLoadError] = React.useState<string | undefined>();
  const [refreshing, setRefreshing] = React.useState(false);
  const [refreshMessage, setRefreshMessage] = React.useState<string | undefined>();

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
    async (source: SourceInfo) => {
      if (!apiState.apiAvailable || !selectedPlugin) {
        return;
      }
      try {
        await apiState.api.enablePluginSource({}, selectedPlugin.name, source.id, !source.enabled);
        fetchSources();
      } catch (err) {
        setLoadError(`Failed to toggle source: ${err instanceof Error ? err.message : String(err)}`);
      }
    },
    [apiState, selectedPlugin, fetchSources],
  );

  const handleDelete = React.useCallback(
    async (source: SourceInfo) => {
      if (!apiState.apiAvailable || !selectedPlugin) {
        return;
      }
      try {
        await apiState.api.deletePluginSource({}, selectedPlugin.name, source.id);
        fetchSources();
      } catch (err) {
        setLoadError(`Failed to delete source: ${err instanceof Error ? err.message : String(err)}`);
      }
    },
    [apiState, selectedPlugin, fetchSources],
  );

  if (!selectedPlugin) {
    return null;
  }

  const hasRefresh = selectedPlugin.management?.refresh;
  const hasSourceManager = selectedPlugin.management?.sourceManager;

  return (
    <>
      <Toolbar>
        <ToolbarContent>
          {hasRefresh && (
            <ToolbarItem>
              <Button
                variant="secondary"
                icon={<SyncIcon />}
                onClick={handleRefreshAll}
                isDisabled={refreshing}
                isLoading={refreshing}
                data-testid="refresh-all-button"
              >
                Refresh all sources
              </Button>
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
              <Th>Name</Th>
              <Th>ID</Th>
              <Th>Type</Th>
              <Th>Status</Th>
              <Th>Entities</Th>
              {hasSourceManager && <Th>Actions</Th>}
            </Tr>
          </Thead>
          <Tbody>
            {sources.length === 0 ? (
              <Tr>
                <Td colSpan={hasSourceManager ? 6 : 5}>No sources configured.</Td>
              </Tr>
            ) : (
              sources.map((source) => (
                <Tr key={source.id} data-testid={`source-row-${source.id}`}>
                  <Td dataLabel="Name">{source.name}</Td>
                  <Td dataLabel="ID">{source.id}</Td>
                  <Td dataLabel="Type">{source.type}</Td>
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
                    <Td dataLabel="Actions">
                      <Button
                        variant="link"
                        onClick={() => handleToggleEnable(source)}
                        data-testid={`toggle-enable-${source.id}`}
                      >
                        {source.enabled ? 'Disable' : 'Enable'}
                      </Button>
                      <Button
                        variant="link"
                        isDanger
                        onClick={() => handleDelete(source)}
                        data-testid={`delete-source-${source.id}`}
                      >
                        Delete
                      </Button>
                    </Td>
                  )}
                </Tr>
              ))
            )}
          </Tbody>
        </Table>
      )}
    </>
  );
};

export default PluginSourcesPage;

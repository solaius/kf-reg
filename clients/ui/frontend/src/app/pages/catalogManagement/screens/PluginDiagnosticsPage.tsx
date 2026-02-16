import * as React from 'react';
import {
  Alert,
  DescriptionList,
  DescriptionListDescription,
  DescriptionListGroup,
  DescriptionListTerm,
  Label,
  Spinner,
  Title,
} from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { CatalogManagementContext } from '~/app/context/catalogManagement/CatalogManagementContext';
import { PluginDiagnostics } from '~/app/catalogManagementTypes';

const PluginDiagnosticsPage: React.FC = () => {
  const { apiState, selectedPlugin } = React.useContext(CatalogManagementContext);

  const [diagnostics, setDiagnostics] = React.useState<PluginDiagnostics | undefined>();
  const [loaded, setLoaded] = React.useState(false);
  const [loadError, setLoadError] = React.useState<string | undefined>();

  React.useEffect(() => {
    if (!apiState.apiAvailable || !selectedPlugin) {
      return;
    }
    setLoaded(false);
    apiState.api
      .getPluginDiagnostics({}, selectedPlugin.name)
      .then((data) => {
        setDiagnostics(data);
        setLoaded(true);
        setLoadError(undefined);
      })
      .catch((err: Error) => {
        setLoadError(err.message);
        setLoaded(true);
      });
  }, [apiState, selectedPlugin]);

  if (!selectedPlugin) {
    return null;
  }

  if (!loaded) {
    return <Spinner size="lg" />;
  }

  if (loadError) {
    return <Alert variant="danger" isInline title={`Failed to load diagnostics: ${loadError}`} />;
  }

  if (!diagnostics) {
    return <Alert variant="info" isInline title="No diagnostic data available." />;
  }

  return (
    <>
      <Title headingLevel="h2">Plugin Diagnostics</Title>

      <DescriptionList isHorizontal>
        <DescriptionListGroup>
          <DescriptionListTerm>Plugin</DescriptionListTerm>
          <DescriptionListDescription>{diagnostics.pluginName}</DescriptionListDescription>
        </DescriptionListGroup>
        {diagnostics.lastRefresh && (
          <DescriptionListGroup>
            <DescriptionListTerm>Last Refresh</DescriptionListTerm>
            <DescriptionListDescription>
              {new Date(diagnostics.lastRefresh).toLocaleString()}
            </DescriptionListDescription>
          </DescriptionListGroup>
        )}
      </DescriptionList>

      {diagnostics.errors && diagnostics.errors.length > 0 && (
        <>
          <Title headingLevel="h3">Errors</Title>
          {diagnostics.errors.map((err, i) => (
            <Alert
              key={i}
              variant="danger"
              isInline
              title={err.message}
              data-testid={`diagnostic-error-${i}`}
            >
              {err.source && <p>Source: {err.source}</p>}
              <p>Time: {new Date(err.time).toLocaleString()}</p>
            </Alert>
          ))}
        </>
      )}

      <Title headingLevel="h3">Source Status</Title>
      <Table aria-label="Source diagnostics" data-testid="diagnostics-table">
        <Thead>
          <Tr>
            <Th>Name</Th>
            <Th>ID</Th>
            <Th>State</Th>
            <Th>Entities</Th>
            <Th>Last Refresh</Th>
            <Th>Error</Th>
          </Tr>
        </Thead>
        <Tbody>
          {!diagnostics.sources || diagnostics.sources.length === 0 ? (
            <Tr>
              <Td colSpan={6}>No sources.</Td>
            </Tr>
          ) : (
            diagnostics.sources.map((src) => (
              <Tr key={src.id} data-testid={`diag-source-${src.id}`}>
                <Td dataLabel="Name">{src.name}</Td>
                <Td dataLabel="ID">{src.id}</Td>
                <Td dataLabel="State">
                  <Label
                    color={
                      src.state === 'available'
                        ? 'green'
                        : src.state === 'error'
                          ? 'red'
                          : 'grey'
                    }
                  >
                    {src.state}
                  </Label>
                </Td>
                <Td dataLabel="Entities">{src.entityCount}</Td>
                <Td dataLabel="Last Refresh">
                  {src.lastRefreshTime ? new Date(src.lastRefreshTime).toLocaleString() : '-'}
                </Td>
                <Td dataLabel="Error">{src.error || '-'}</Td>
              </Tr>
            ))
          )}
        </Tbody>
      </Table>
    </>
  );
};

export default PluginDiagnosticsPage;

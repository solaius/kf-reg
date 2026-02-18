import * as React from 'react';
import {
  Button,
  Card,
  CardBody,
  CardTitle,
  EmptyState,
} from '@patternfly/react-core';
import {
  Table,
  Thead,
  Tr,
  Th,
  Tbody,
  Td,
} from '@patternfly/react-table';
import { VersionResponse, VersionListResponse } from '~/app/types/governance';
import { listVersions, createVersion } from '~/app/api/governance/service';

type VersionsPanelProps = {
  plugin: string;
  kind: string;
  name: string;
};

const VersionsPanel: React.FC<VersionsPanelProps> = ({ plugin, kind, name }) => {
  const [versions, setVersions] = React.useState<VersionResponse[]>([]);
  const [loaded, setLoaded] = React.useState(false);

  const fetchVersions = React.useCallback(() => {
    const fetcher = listVersions(plugin, kind, name);
    fetcher({})
      .then((result: VersionListResponse) => {
        setVersions(result.versions || []);
        setLoaded(true);
      })
      .catch(() => setLoaded(true));
  }, [plugin, kind, name]);

  React.useEffect(() => {
    fetchVersions();
  }, [fetchVersions]);

  const handleCreate = () => {
    const label = prompt('Enter version label (e.g. v1.0):');
    if (!label) {
      return;
    }
    const creator = createVersion(plugin, kind, name, label);
    creator({}).then(() => fetchVersions());
  };

  if (!loaded) {
    return null;
  }

  return (
    <Card>
      <CardTitle>
        Versions{' '}
        <Button variant="secondary" size="sm" onClick={handleCreate}>
          Create Version
        </Button>
      </CardTitle>
      <CardBody>
        {versions.length === 0 ? (
          <EmptyState titleText="No versions" headingLevel="h4">
            <p>No versions have been created for this asset.</p>
          </EmptyState>
        ) : (
          <Table aria-label="Versions" variant="compact">
            <Thead>
              <Tr>
                <Th>Version ID</Th>
                <Th>Label</Th>
                <Th>Created By</Th>
                <Th>Created At</Th>
              </Tr>
            </Thead>
            <Tbody>
              {versions.map((v) => (
                <Tr key={v.versionId}>
                  <Td dataLabel="Version ID">{v.versionId}</Td>
                  <Td dataLabel="Label">{v.versionLabel}</Td>
                  <Td dataLabel="Created By">{v.createdBy}</Td>
                  <Td dataLabel="Created At">
                    {new Date(v.createdAt).toLocaleString()}
                  </Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </CardBody>
    </Card>
  );
};

export default VersionsPanel;

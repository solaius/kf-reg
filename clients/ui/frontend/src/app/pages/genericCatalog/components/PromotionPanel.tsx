import * as React from 'react';
import {
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
import { BindingResponse, BindingsResponse } from '~/app/types/governance';
import { listBindings } from '~/app/api/governance/service';

type PromotionPanelProps = {
  plugin: string;
  kind: string;
  name: string;
};

const PromotionPanel: React.FC<PromotionPanelProps> = ({ plugin, kind, name }) => {
  const [bindings, setBindings] = React.useState<BindingResponse[]>([]);
  const [loaded, setLoaded] = React.useState(false);

  React.useEffect(() => {
    const fetcher = listBindings(plugin, kind, name);
    fetcher({})
      .then((result: BindingsResponse) => {
        setBindings(result.bindings || []);
        setLoaded(true);
      })
      .catch(() => setLoaded(true));
  }, [plugin, kind, name]);

  if (!loaded) {
    return null;
  }

  return (
    <Card>
      <CardTitle>Environment Bindings</CardTitle>
      <CardBody>
        {bindings.length === 0 ? (
          <EmptyState titleText="No bindings" headingLevel="h4">
            <p>No environment bindings have been configured for this asset.</p>
          </EmptyState>
        ) : (
          <Table aria-label="Environment bindings" variant="compact">
            <Thead>
              <Tr>
                <Th>Environment</Th>
                <Th>Version ID</Th>
                <Th>Bound By</Th>
                <Th>Bound At</Th>
                <Th>Previous Version</Th>
              </Tr>
            </Thead>
            <Tbody>
              {bindings.map((b) => (
                <Tr key={b.environment}>
                  <Td dataLabel="Environment">{b.environment}</Td>
                  <Td dataLabel="Version ID">{b.versionId}</Td>
                  <Td dataLabel="Bound By">{b.boundBy}</Td>
                  <Td dataLabel="Bound At">
                    {new Date(b.boundAt).toLocaleString()}
                  </Td>
                  <Td dataLabel="Previous Version">{b.previousVersionId || '-'}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
      </CardBody>
    </Card>
  );
};

export default PromotionPanel;

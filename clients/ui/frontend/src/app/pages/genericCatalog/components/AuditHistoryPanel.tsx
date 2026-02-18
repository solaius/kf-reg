import * as React from 'react';
import {
  Button,
  Card,
  CardBody,
  CardTitle,
  EmptyState,
  Flex,
  FlexItem,
} from '@patternfly/react-core';
import {
  Table,
  Thead,
  Tr,
  Th,
  Tbody,
  Td,
} from '@patternfly/react-table';
import { AuditEvent, AuditEventList } from '~/app/types/governance';
import { getGovernanceHistory } from '~/app/api/governance/service';

type AuditHistoryPanelProps = {
  plugin: string;
  kind: string;
  name: string;
};

const AuditHistoryPanel: React.FC<AuditHistoryPanelProps> = ({ plugin, kind, name }) => {
  const [events, setEvents] = React.useState<AuditEvent[]>([]);
  const [loaded, setLoaded] = React.useState(false);
  const [nextPageToken, setNextPageToken] = React.useState<string | undefined>();

  const fetchHistory = React.useCallback(
    (pageToken?: string) => {
      const params: Record<string, unknown> = { pageSize: 20 };
      if (pageToken) {
        params.pageToken = pageToken;
      }
      const fetcher = getGovernanceHistory(plugin, kind, name, params);
      fetcher({})
        .then((result: AuditEventList) => {
          if (pageToken) {
            setEvents((prev) => [...prev, ...(result.events || [])]);
          } else {
            setEvents(result.events || []);
          }
          setNextPageToken(result.nextPageToken);
          setLoaded(true);
        })
        .catch(() => setLoaded(true));
    },
    [plugin, kind, name],
  );

  React.useEffect(() => {
    fetchHistory();
  }, [fetchHistory]);

  if (!loaded) {
    return null;
  }

  return (
    <Card>
      <CardTitle>Audit History</CardTitle>
      <CardBody>
        {events.length === 0 ? (
          <EmptyState titleText="No audit events" headingLevel="h4">
            <p>No audit events have been recorded for this asset.</p>
          </EmptyState>
        ) : (
          <>
            <Table aria-label="Audit history" variant="compact">
              <Thead>
                <Tr>
                  <Th>Event</Th>
                  <Th>Actor</Th>
                  <Th>Action</Th>
                  <Th>Outcome</Th>
                  <Th>Reason</Th>
                  <Th>Time</Th>
                </Tr>
              </Thead>
              <Tbody>
                {events.map((e) => (
                  <Tr key={e.id}>
                    <Td dataLabel="Event">{e.eventType}</Td>
                    <Td dataLabel="Actor">{e.actor}</Td>
                    <Td dataLabel="Action">{e.action || '-'}</Td>
                    <Td dataLabel="Outcome">{e.outcome}</Td>
                    <Td dataLabel="Reason">{e.reason || '-'}</Td>
                    <Td dataLabel="Time">{new Date(e.createdAt).toLocaleString()}</Td>
                  </Tr>
                ))}
              </Tbody>
            </Table>
            {nextPageToken && (
              <Flex justifyContent={{ default: 'justifyContentCenter' }}>
                <FlexItem>
                  <Button variant="link" onClick={() => fetchHistory(nextPageToken)}>
                    Load more
                  </Button>
                </FlexItem>
              </Flex>
            )}
          </>
        )}
      </CardBody>
    </Card>
  );
};

export default AuditHistoryPanel;

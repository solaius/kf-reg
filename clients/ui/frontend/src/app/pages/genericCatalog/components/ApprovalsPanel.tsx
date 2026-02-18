import * as React from 'react';
import {
  Button,
  Card,
  CardBody,
  CardTitle,
  EmptyState,
  Flex,
  FlexItem,
  Label,
} from '@patternfly/react-core';
import {
  Table,
  Thead,
  Tr,
  Th,
  Tbody,
  Td,
} from '@patternfly/react-table';
import { ApprovalRequest, ApprovalRequestList } from '~/app/types/governance';
import { listApprovals, submitDecision, cancelApproval } from '~/app/api/governance/service';
import { useNotification } from '~/app/hooks/useNotification';

type ApprovalsPanelProps = {
  plugin?: string;
  kind?: string;
  name?: string;
};

const statusColors: Record<string, 'green' | 'yellow' | 'red' | 'grey' | 'orange'> = {
  pending: 'yellow',
  approved: 'green',
  denied: 'red',
  canceled: 'grey',
  expired: 'orange',
};

const ApprovalsPanel: React.FC<ApprovalsPanelProps> = ({ plugin, kind, name }) => {
  const notification = useNotification();
  const [approvals, setApprovals] = React.useState<ApprovalRequest[]>([]);
  const [loaded, setLoaded] = React.useState(false);

  const fetchApprovals = React.useCallback(() => {
    const params: Record<string, unknown> = {};
    if (plugin && kind && name) {
      params.assetUid = `${plugin}:${kind}:${name}`;
    }
    const fetcher = listApprovals(params);
    fetcher({})
      .then((result: ApprovalRequestList) => {
        setApprovals(result.requests || []);
        setLoaded(true);
      })
      .catch(() => setLoaded(true));
  }, [plugin, kind, name]);

  React.useEffect(() => {
    fetchApprovals();
  }, [fetchApprovals]);

  const handleApprove = async (id: string) => {
    const comment = prompt('Approval comment (optional):') || '';
    try {
      const executor = submitDecision(id, 'approve', comment);
      await executor({});
      notification.success('Approval submitted');
      fetchApprovals();
    } catch (err) {
      notification.error('Failed to approve', (err as Error).message);
    }
  };

  const handleReject = async (id: string) => {
    const comment = prompt('Rejection reason:') || '';
    try {
      const executor = submitDecision(id, 'deny', comment);
      await executor({});
      notification.success('Rejection submitted');
      fetchApprovals();
    } catch (err) {
      notification.error('Failed to reject', (err as Error).message);
    }
  };

  const handleCancel = async (id: string) => {
    const reason = prompt('Cancellation reason (optional):') || '';
    try {
      const executor = cancelApproval(id, reason);
      await executor({});
      notification.success('Approval canceled');
      fetchApprovals();
    } catch (err) {
      notification.error('Failed to cancel', (err as Error).message);
    }
  };

  if (!loaded) {
    return null;
  }

  return (
    <Card>
      <CardTitle>Approvals</CardTitle>
      <CardBody>
        {approvals.length === 0 ? (
          <EmptyState titleText="No approval requests" headingLevel="h4">
            <p>There are no approval requests for this asset.</p>
          </EmptyState>
        ) : (
          <Table aria-label="Approval requests" variant="compact">
            <Thead>
              <Tr>
                <Th>Action</Th>
                <Th>Requester</Th>
                <Th>Status</Th>
                <Th>Policy</Th>
                <Th>Created</Th>
                <Th>Actions</Th>
              </Tr>
            </Thead>
            <Tbody>
              {approvals.map((a) => (
                <Tr key={a.id}>
                  <Td dataLabel="Action">{a.action}</Td>
                  <Td dataLabel="Requester">{a.requester}</Td>
                  <Td dataLabel="Status">
                    <Label color={statusColors[a.status] || 'grey'}>{a.status}</Label>
                  </Td>
                  <Td dataLabel="Policy">{a.policyId}</Td>
                  <Td dataLabel="Created">{new Date(a.createdAt).toLocaleString()}</Td>
                  <Td dataLabel="Actions">
                    {a.status === 'pending' && (
                      <Flex gap={{ default: 'gapSm' }}>
                        <FlexItem>
                          <Button variant="primary" size="sm" onClick={() => handleApprove(a.id)}>
                            Approve
                          </Button>
                        </FlexItem>
                        <FlexItem>
                          <Button variant="danger" size="sm" onClick={() => handleReject(a.id)}>
                            Reject
                          </Button>
                        </FlexItem>
                        <FlexItem>
                          <Button variant="secondary" size="sm" onClick={() => handleCancel(a.id)}>
                            Cancel
                          </Button>
                        </FlexItem>
                      </Flex>
                    )}
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

export default ApprovalsPanel;

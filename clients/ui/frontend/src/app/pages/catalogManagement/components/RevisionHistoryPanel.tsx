import * as React from 'react';
import {
  Button,
  Panel,
  PanelMain,
  PanelMainBody,
  PanelHeader,
  DataList,
  DataListCell,
  DataListItem,
  DataListItemCells,
  DataListItemRow,
  Spinner,
  Title,
} from '@patternfly/react-core';
import { ConfigRevision } from '~/app/catalogManagementTypes';

type RevisionHistoryPanelProps = {
  revisions: ConfigRevision[];
  loading: boolean;
  onRollback: (version: string) => void;
};

const formatRelativeTime = (timestamp: string): string => {
  const date = new Date(timestamp);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffSec = Math.floor(diffMs / 1000);
  const diffMin = Math.floor(diffSec / 60);
  const diffHour = Math.floor(diffMin / 60);
  const diffDay = Math.floor(diffHour / 24);

  if (diffSec < 60) {
    return 'just now';
  }
  if (diffMin < 60) {
    return `${diffMin}m ago`;
  }
  if (diffHour < 24) {
    return `${diffHour}h ago`;
  }
  if (diffDay < 30) {
    return `${diffDay}d ago`;
  }
  return date.toLocaleDateString();
};

const formatSize = (bytes: number): string => {
  if (bytes < 1024) {
    return `${bytes} B`;
  }
  if (bytes < 1024 * 1024) {
    return `${(bytes / 1024).toFixed(1)} KB`;
  }
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`;
};

const truncateVersion = (version: string): string =>
  version.length > 12 ? `${version.slice(0, 12)}...` : version;

const RevisionHistoryPanel: React.FC<RevisionHistoryPanelProps> = ({
  revisions,
  loading,
  onRollback,
}) => (
  <Panel data-testid="revision-history-panel">
    <PanelHeader>
      <Title headingLevel="h3" size="md">
        Revision history
      </Title>
    </PanelHeader>
    <PanelMain>
      <PanelMainBody>
        {loading ? (
          <Spinner size="md" />
        ) : revisions.length === 0 ? (
          <p>No revisions available.</p>
        ) : (
          <DataList aria-label="Revision history" isCompact data-testid="revision-list">
            {revisions.map((rev) => (
              <DataListItem key={rev.version} aria-labelledby={`rev-${rev.version}`}>
                <DataListItemRow>
                  <DataListItemCells
                    dataListCells={[
                      <DataListCell key="version" width={2}>
                        <code id={`rev-${rev.version}`} title={rev.version}>
                          {truncateVersion(rev.version)}
                        </code>
                      </DataListCell>,
                      <DataListCell key="timestamp" width={2}>
                        <span title={rev.timestamp}>{formatRelativeTime(rev.timestamp)}</span>
                      </DataListCell>,
                      <DataListCell key="size" width={1}>
                        {formatSize(rev.size)}
                      </DataListCell>,
                      <DataListCell key="action" width={1}>
                        <Button
                          variant="link"
                          size="sm"
                          onClick={() => onRollback(rev.version)}
                          data-testid={`rollback-${rev.version}`}
                        >
                          Rollback
                        </Button>
                      </DataListCell>,
                    ]}
                  />
                </DataListItemRow>
              </DataListItem>
            ))}
          </DataList>
        )}
      </PanelMainBody>
    </PanelMain>
  </Panel>
);

export default RevisionHistoryPanel;

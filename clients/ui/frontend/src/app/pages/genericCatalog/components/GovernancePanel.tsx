import * as React from 'react';
import {
  Card,
  CardBody,
  CardTitle,
  DescriptionList,
  DescriptionListDescription,
  DescriptionListGroup,
  DescriptionListTerm,
  Label,
  LabelGroup,
  Stack,
  StackItem,
} from '@patternfly/react-core';
import { GovernanceOverlay } from '~/app/types/governance';
import LifecycleBadge from './LifecycleBadge';

type GovernancePanelProps = {
  governance: GovernanceOverlay;
};

const GovernancePanel: React.FC<GovernancePanelProps> = ({ governance }) => (
  <Stack hasGutter>
    <StackItem>
      <Card>
        <CardTitle>Lifecycle</CardTitle>
        <CardBody>
          <DescriptionList>
            <DescriptionListGroup>
              <DescriptionListTerm>State</DescriptionListTerm>
              <DescriptionListDescription>
                {governance.lifecycle ? (
                  <LifecycleBadge state={governance.lifecycle.state} />
                ) : (
                  '-'
                )}
              </DescriptionListDescription>
            </DescriptionListGroup>
            {governance.lifecycle?.reason && (
              <DescriptionListGroup>
                <DescriptionListTerm>Reason</DescriptionListTerm>
                <DescriptionListDescription>
                  {governance.lifecycle.reason}
                </DescriptionListDescription>
              </DescriptionListGroup>
            )}
            {governance.lifecycle?.changedBy && (
              <DescriptionListGroup>
                <DescriptionListTerm>Changed By</DescriptionListTerm>
                <DescriptionListDescription>
                  {governance.lifecycle.changedBy}
                </DescriptionListDescription>
              </DescriptionListGroup>
            )}
            {governance.lifecycle?.changedAt && (
              <DescriptionListGroup>
                <DescriptionListTerm>Changed At</DescriptionListTerm>
                <DescriptionListDescription>
                  {new Date(governance.lifecycle.changedAt).toLocaleString()}
                </DescriptionListDescription>
              </DescriptionListGroup>
            )}
          </DescriptionList>
        </CardBody>
      </Card>
    </StackItem>

    <StackItem>
      <Card>
        <CardTitle>Ownership</CardTitle>
        <CardBody>
          <DescriptionList>
            <DescriptionListGroup>
              <DescriptionListTerm>Owner</DescriptionListTerm>
              <DescriptionListDescription>
                {governance.owner?.displayName || governance.owner?.principal || '-'}
              </DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
              <DescriptionListTerm>Team</DescriptionListTerm>
              <DescriptionListDescription>
                {governance.team?.name || '-'}
              </DescriptionListDescription>
            </DescriptionListGroup>
          </DescriptionList>
        </CardBody>
      </Card>
    </StackItem>

    <StackItem>
      <Card>
        <CardTitle>Risk and Compliance</CardTitle>
        <CardBody>
          <DescriptionList>
            <DescriptionListGroup>
              <DescriptionListTerm>SLA Tier</DescriptionListTerm>
              <DescriptionListDescription>
                {governance.sla?.tier || '-'}
              </DescriptionListDescription>
            </DescriptionListGroup>
            <DescriptionListGroup>
              <DescriptionListTerm>Risk Level</DescriptionListTerm>
              <DescriptionListDescription>
                {governance.risk?.level || '-'}
              </DescriptionListDescription>
            </DescriptionListGroup>
            {governance.compliance?.tags && governance.compliance.tags.length > 0 && (
              <DescriptionListGroup>
                <DescriptionListTerm>Compliance Tags</DescriptionListTerm>
                <DescriptionListDescription>
                  <LabelGroup>
                    {governance.compliance.tags.map((tag) => (
                      <Label key={tag}>{tag}</Label>
                    ))}
                  </LabelGroup>
                </DescriptionListDescription>
              </DescriptionListGroup>
            )}
          </DescriptionList>
        </CardBody>
      </Card>
    </StackItem>
  </Stack>
);

export default GovernancePanel;

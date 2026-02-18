import * as React from 'react';
import { Label } from '@patternfly/react-core';
import { LifecycleState } from '~/app/types/governance';

type LifecycleBadgeProps = {
  state: LifecycleState | string;
};

const stateColorMap: Record<string, 'yellow' | 'green' | 'orange' | 'grey'> = {
  draft: 'yellow',
  approved: 'green',
  deprecated: 'orange',
  archived: 'grey',
};

const LifecycleBadge: React.FC<LifecycleBadgeProps> = ({ state }) => {
  const color = stateColorMap[state] || 'grey';
  return <Label color={color}>{state}</Label>;
};

export default LifecycleBadge;

import * as React from 'react';
import { Button, Flex, FlexItem } from '@patternfly/react-core';
import { ActionDefinition } from '~/app/types/capabilities';

type GenericActionBarProps = {
  actions: ActionDefinition[];
  onActionClick: (actionId: string) => void;
};

const GenericActionBar: React.FC<GenericActionBarProps> = ({ actions, onActionClick }) => {
  if (actions.length === 0) {
    return null;
  }

  return (
    <Flex gap={{ default: 'gapSm' }}>
      {actions.map((action) => (
        <FlexItem key={action.id}>
          <Button
            variant={action.destructive ? 'danger' : 'secondary'}
            onClick={() => onActionClick(action.id)}
          >
            {action.displayName}
          </Button>
        </FlexItem>
      ))}
    </Flex>
  );
};

export default GenericActionBar;

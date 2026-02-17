import * as React from 'react';
import {
  Table,
  Thead,
  Tr,
  Th,
  Tbody,
  Td,
} from '@patternfly/react-table';
import { Button } from '@patternfly/react-core';
import { EntityCapabilities } from '~/app/types/capabilities';
import { GenericEntity } from '~/app/types/asset';
import { getFieldValue, formatFieldValue } from '~/app/pages/genericCatalog/utils';

type GenericListViewProps = {
  entity: EntityCapabilities;
  entities: GenericEntity[];
  onEntityClick: (name: string) => void;
};

const GenericListView: React.FC<GenericListViewProps> = ({
  entity,
  entities,
  onEntityClick,
}) => {
  const columns = entity.fields.columns;
  const nameField = entity.uiHints?.nameField || 'name';

  return (
    <Table aria-label={`${entity.displayName} list`} variant="compact">
      <Thead>
        <Tr>
          {columns.map((col) => (
            <Th key={col.path} width={col.width ? parseInt(col.width, 10) as 10 | 15 | 20 | 25 | 30 | 35 | 40 | 45 | 50 | 60 | 70 | 80 | 90 | 100 : undefined}>
              {col.displayName}
            </Th>
          ))}
        </Tr>
      </Thead>
      <Tbody>
        {entities.map((item, rowIndex) => {
          const entityName = String(getFieldValue(item, nameField) || `item-${rowIndex}`);
          return (
            <Tr key={entityName}>
              {columns.map((col, colIndex) => {
                const value = getFieldValue(item, col.path);
                const isNameColumn = col.path === nameField || colIndex === 0;
                return (
                  <Td key={col.path} dataLabel={col.displayName}>
                    {isNameColumn ? (
                      <Button
                        variant="link"
                        isInline
                        onClick={() => onEntityClick(entityName)}
                      >
                        {formatFieldValue(value, col.type)}
                      </Button>
                    ) : (
                      formatFieldValue(value, col.type)
                    )}
                  </Td>
                );
              })}
            </Tr>
          );
        })}
      </Tbody>
    </Table>
  );
};

export default GenericListView;

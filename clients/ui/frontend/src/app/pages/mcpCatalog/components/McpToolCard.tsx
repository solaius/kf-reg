import * as React from 'react';
import { ExpandableSection, Label } from '@patternfly/react-core';
import { Table, Thead, Tr, Th, Tbody, Td } from '@patternfly/react-table';
import { McpTool } from '~/app/mcpCatalogTypes';

type McpToolCardProps = {
  tool: McpTool;
};

const accessTypeBadge = (
  accessType: string,
): { label: string; color: 'green' | 'orange' | 'red' } => {
  switch (accessType) {
    case 'read_only':
      return { label: 'Read Only', color: 'green' };
    case 'read_write':
      return { label: 'Read/Write', color: 'orange' };
    case 'destructive':
      return { label: 'Destructive', color: 'red' };
    default:
      return { label: accessType, color: 'orange' };
  }
};

const McpToolCard: React.FC<McpToolCardProps> = ({ tool }) => {
  const [isExpanded, setIsExpanded] = React.useState(false);
  const badge = accessTypeBadge(tool.accessType);

  const toggleContent = (
    <span>
      <strong>{tool.name}</strong>{' '}
      <Label color={badge.color} isCompact className="pf-v6-u-ml-sm">
        {badge.label}
      </Label>{' '}
      <span className="pf-v6-u-color-200 pf-v6-u-ml-sm">
        {tool.description.length > 80
          ? `${tool.description.substring(0, 80)}...`
          : tool.description}
      </span>
    </span>
  );

  return (
    <ExpandableSection
      toggleContent={toggleContent}
      isExpanded={isExpanded}
      onToggle={(_event, expanded) => setIsExpanded(expanded)}
      className="pf-v6-u-mb-sm"
    >
      <div className="pf-v6-u-ml-lg pf-v6-u-mb-md">
        <p className="pf-v6-u-mb-md">{tool.description}</p>
        {tool.parameters && tool.parameters.length > 0 && (
          <Table aria-label={`${tool.name} parameters`} variant="compact" borders>
            <Thead>
              <Tr>
                <Th>Name</Th>
                <Th>Type</Th>
                <Th>Required</Th>
                <Th>Description</Th>
              </Tr>
            </Thead>
            <Tbody>
              {tool.parameters.map((param) => (
                <Tr key={param.name}>
                  <Td dataLabel="Name">
                    <code>{param.name}</code>
                  </Td>
                  <Td dataLabel="Type">
                    <code>{param.type}</code>
                  </Td>
                  <Td dataLabel="Required">{param.required ? 'Yes' : 'No'}</Td>
                  <Td dataLabel="Description">{param.description}</Td>
                </Tr>
              ))}
            </Tbody>
          </Table>
        )}
        {(!tool.parameters || tool.parameters.length === 0) && (
          <p className="pf-v6-u-color-200">No parameters</p>
        )}
      </div>
    </ExpandableSection>
  );
};

export default McpToolCard;

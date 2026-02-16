import React from 'react';
import {
  Button,
  Card,
  CardBody,
  CardFooter,
  CardHeader,
  CardTitle,
  Flex,
  FlexItem,
  Label,
  LabelGroup,
  Truncate,
} from '@patternfly/react-core';
import { CheckCircleIcon, CubesIcon } from '@patternfly/react-icons';
import { Link } from 'react-router-dom';
import { McpServer } from '~/app/mcpCatalogTypes';
import { mcpServerDetailUrl } from '~/app/routes/mcpCatalog/mcpCatalog';

type McpServerCardProps = {
  server: McpServer;
};

const McpServerCard: React.FC<McpServerCardProps> = ({ server }) => (
  <Card isFullHeight data-testid={`mcp-server-card-${server.name}`}>
    <CardHeader>
      <CardTitle>
        <Flex alignItems={{ default: 'alignItemsFlexStart' }} className="pf-v6-u-mb-md">
          <FlexItem>
            <CubesIcon
              style={{
                fontSize: '2rem',
                color: 'var(--pf-t--global--color--brand--default)',
              }}
            />
          </FlexItem>
          <FlexItem align={{ default: 'alignRight' }}>
            {server.verified && (
              <Label color="green" icon={<CheckCircleIcon />}>
                Verified
              </Label>
            )}
            {server.certified && (
              <Label color="purple" className="pf-v6-u-ml-sm">
                Certified
              </Label>
            )}
          </FlexItem>
        </Flex>
        <Link to={mcpServerDetailUrl(server.name)}>
          <Button
            data-testid="mcp-server-detail-link"
            variant="link"
            tabIndex={-1}
            isInline
            style={{
              fontSize: 'var(--pf-t--global--font--size--body--default)',
              fontWeight: 'var(--pf-t--global--font--weight--body--bold)',
            }}
          >
            <Truncate content={server.name} position="middle" tooltipPosition="top" />
          </Button>
        </Link>
      </CardTitle>
    </CardHeader>
    <CardBody>
      {server.description && (
        <Truncate content={server.description} tooltipPosition="top" />
      )}
      <Flex className="pf-v6-u-mt-md" gap={{ default: 'gapMd' }}>
        <FlexItem>
          <strong>{server.toolCount ?? 0}</strong> tools
        </FlexItem>
        <FlexItem>
          <strong>{server.resourceCount ?? 0}</strong> resources
        </FlexItem>
        <FlexItem>
          <strong>{server.promptCount ?? 0}</strong> prompts
        </FlexItem>
      </Flex>
    </CardBody>
    <CardFooter>
      <LabelGroup>
        {server.deploymentMode && (
          <Label color={server.deploymentMode === 'local' ? 'blue' : 'orange'}>
            {server.deploymentMode}
          </Label>
        )}
        {server.provider && <Label>{server.provider}</Label>}
        {server.category && <Label color="grey">{server.category}</Label>}
        {server.license && <Label color="teal">{server.license}</Label>}
      </LabelGroup>
    </CardFooter>
  </Card>
);

export default McpServerCard;

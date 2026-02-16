import * as React from 'react';
import {
  Alert,
  Breadcrumb,
  BreadcrumbItem,
  DescriptionList,
  DescriptionListDescription,
  DescriptionListGroup,
  DescriptionListTerm,
  Flex,
  FlexItem,
  Label,
  PageSection,
  Spinner,
  Tab,
  Tabs,
  TabTitleText,
  Title,
} from '@patternfly/react-core';
import { CheckCircleIcon, CubesIcon } from '@patternfly/react-icons';
import { Link, useParams } from 'react-router-dom';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import { mcpCatalogUrl, MCP_CATALOG_PAGE_TITLE } from '~/app/routes/mcpCatalog/mcpCatalog';

const McpServerDetailPage: React.FC = () => {
  const { serverName } = useParams<{ serverName: string }>();
  const { mcpServers, mcpServersLoaded } = React.useContext(McpCatalogContext);
  const [activeTabKey, setActiveTabKey] = React.useState(0);

  const server = React.useMemo(
    () => mcpServers.find((s) => s.name === serverName),
    [mcpServers, serverName],
  );

  if (!mcpServersLoaded) {
    return (
      <PageSection>
        <Spinner />
      </PageSection>
    );
  }

  if (!server) {
    return (
      <PageSection>
        <Alert variant="warning" title={`MCP server "${serverName}" not found`} isInline />
      </PageSection>
    );
  }

  return (
    <>
      <PageSection>
        <Breadcrumb>
          <BreadcrumbItem>
            <Link to={mcpCatalogUrl()}>{MCP_CATALOG_PAGE_TITLE}</Link>
          </BreadcrumbItem>
          <BreadcrumbItem isActive>{server.name}</BreadcrumbItem>
        </Breadcrumb>
      </PageSection>
      <PageSection>
        <Flex alignItems={{ default: 'alignItemsCenter' }} gap={{ default: 'gapMd' }}>
          <FlexItem>
            <CubesIcon
              style={{
                fontSize: '2.5rem',
                color: 'var(--pf-t--global--color--brand--default)',
              }}
            />
          </FlexItem>
          <FlexItem>
            <Title headingLevel="h1">{server.name}</Title>
            {server.provider && (
              <span className="pf-v6-u-color-200">by {server.provider}</span>
            )}
          </FlexItem>
          <FlexItem align={{ default: 'alignRight' }}>
            {server.verified && (
              <Label color="green" icon={<CheckCircleIcon />} className="pf-v6-u-mr-sm">
                Verified
              </Label>
            )}
            {server.certified && <Label color="purple">Certified</Label>}
          </FlexItem>
        </Flex>
        {server.description && <p className="pf-v6-u-mt-md">{server.description}</p>}
      </PageSection>
      <PageSection>
        <Tabs
          activeKey={activeTabKey}
          onSelect={(_event, key) => setActiveTabKey(key as number)}
        >
          <Tab eventKey={0} title={<TabTitleText>Overview</TabTitleText>}>
            <PageSection padding={{ default: 'noPadding' }}>
              <DescriptionList className="pf-v6-u-mt-lg">
                <DescriptionListGroup>
                  <DescriptionListTerm>Server URL</DescriptionListTerm>
                  <DescriptionListDescription>{server.serverUrl}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Deployment Mode</DescriptionListTerm>
                  <DescriptionListDescription>
                    <Label color={server.deploymentMode === 'local' ? 'blue' : 'orange'}>
                      {server.deploymentMode || 'unknown'}
                    </Label>
                  </DescriptionListDescription>
                </DescriptionListGroup>
                {server.image && (
                  <DescriptionListGroup>
                    <DescriptionListTerm>Container Image</DescriptionListTerm>
                    <DescriptionListDescription>{server.image}</DescriptionListDescription>
                  </DescriptionListGroup>
                )}
                {server.endpoint && (
                  <DescriptionListGroup>
                    <DescriptionListTerm>Remote Endpoint</DescriptionListTerm>
                    <DescriptionListDescription>{server.endpoint}</DescriptionListDescription>
                  </DescriptionListGroup>
                )}
                <DescriptionListGroup>
                  <DescriptionListTerm>Supported Transports</DescriptionListTerm>
                  <DescriptionListDescription>
                    {server.supportedTransports || server.transportType || 'N/A'}
                  </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>License</DescriptionListTerm>
                  <DescriptionListDescription>
                    {server.license || 'Not specified'}
                  </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Tools</DescriptionListTerm>
                  <DescriptionListDescription>{server.toolCount ?? 0}</DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Resources</DescriptionListTerm>
                  <DescriptionListDescription>
                    {server.resourceCount ?? 0}
                  </DescriptionListDescription>
                </DescriptionListGroup>
                <DescriptionListGroup>
                  <DescriptionListTerm>Prompts</DescriptionListTerm>
                  <DescriptionListDescription>
                    {server.promptCount ?? 0}
                  </DescriptionListDescription>
                </DescriptionListGroup>
                {server.category && (
                  <DescriptionListGroup>
                    <DescriptionListTerm>Category</DescriptionListTerm>
                    <DescriptionListDescription>{server.category}</DescriptionListDescription>
                  </DescriptionListGroup>
                )}
              </DescriptionList>
            </PageSection>
          </Tab>
        </Tabs>
      </PageSection>
    </>
  );
};

export default McpServerDetailPage;

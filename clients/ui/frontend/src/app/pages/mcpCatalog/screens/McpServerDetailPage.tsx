import * as React from 'react';
import {
  Alert,
  Breadcrumb,
  BreadcrumbItem,
  Card,
  CardBody,
  CardTitle,
  Flex,
  FlexItem,
  Grid,
  GridItem,
  Label,
  PageSection,
  Pagination,
  SearchInput,
  Spinner,
  Title,
} from '@patternfly/react-core';
import { CheckCircleIcon, CubesIcon } from '@patternfly/react-icons';
import { Link, useParams } from 'react-router-dom';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import { mcpCatalogUrl, MCP_CATALOG_PAGE_TITLE } from '~/app/routes/mcpCatalog/mcpCatalog';
import McpToolCard from '~/app/pages/mcpCatalog/components/McpToolCard';
import McpReadme from '~/app/pages/mcpCatalog/components/McpReadme';
import McpServerDetailsSidebar from '~/app/pages/mcpCatalog/components/McpServerDetailsSidebar';

const TOOLS_PER_PAGE = 5;

const McpServerDetailPage: React.FC = () => {
  const { serverName } = useParams<{ serverName: string }>();
  const { mcpServers, mcpServersLoaded } = React.useContext(McpCatalogContext);

  const [toolSearchTerm, setToolSearchTerm] = React.useState('');
  const [toolPage, setToolPage] = React.useState(1);

  const server = React.useMemo(
    () => mcpServers.find((s) => s.name === serverName),
    [mcpServers, serverName],
  );

  const filteredTools = React.useMemo(() => {
    if (!server?.tools) {
      return [];
    }
    if (!toolSearchTerm) {
      return server.tools;
    }
    const term = toolSearchTerm.toLowerCase();
    return server.tools.filter(
      (t) =>
        t.name.toLowerCase().includes(term) || t.description.toLowerCase().includes(term),
    );
  }, [server?.tools, toolSearchTerm]);

  const pagedTools = React.useMemo(() => {
    const start = (toolPage - 1) * TOOLS_PER_PAGE;
    return filteredTools.slice(start, start + TOOLS_PER_PAGE);
  }, [filteredTools, toolPage]);

  // Reset to page 1 when search changes
  React.useEffect(() => {
    setToolPage(1);
  }, [toolSearchTerm]);

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

  const toolCount = server.tools?.length ?? server.toolCount ?? 0;

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
      </PageSection>
      <PageSection>
        <Grid hasGutter>
          <GridItem span={8}>
            {/* Description */}
            {server.description && (
              <Card className="pf-v6-u-mb-lg">
                <CardTitle>Description</CardTitle>
                <CardBody>{server.description}</CardBody>
              </Card>
            )}

            {/* Tools */}
            {toolCount > 0 && (
              <Card className="pf-v6-u-mb-lg">
                <CardTitle>Tools ({toolCount})</CardTitle>
                <CardBody>
                  <SearchInput
                    placeholder="Filter tools by name or description..."
                    value={toolSearchTerm}
                    onChange={(_event, value) => setToolSearchTerm(value)}
                    onClear={() => setToolSearchTerm('')}
                    className="pf-v6-u-mb-md"
                  />
                  {pagedTools.length > 0 ? (
                    pagedTools.map((tool) => <McpToolCard key={tool.name} tool={tool} />)
                  ) : (
                    <p className="pf-v6-u-color-200">
                      No tools match the filter &quot;{toolSearchTerm}&quot;
                    </p>
                  )}
                  {filteredTools.length > TOOLS_PER_PAGE && (
                    <Pagination
                      itemCount={filteredTools.length}
                      perPage={TOOLS_PER_PAGE}
                      page={toolPage}
                      onSetPage={(_event, page) => setToolPage(page)}
                      isCompact
                      className="pf-v6-u-mt-md"
                    />
                  )}
                </CardBody>
              </Card>
            )}

            {/* README */}
            {server.readme && (
              <Card className="pf-v6-u-mb-lg">
                <CardTitle>README</CardTitle>
                <CardBody>
                  <McpReadme content={server.readme} />
                </CardBody>
              </Card>
            )}
          </GridItem>

          <GridItem span={4}>
            <McpServerDetailsSidebar server={server} />
          </GridItem>
        </Grid>
      </PageSection>
    </>
  );
};

export default McpServerDetailPage;

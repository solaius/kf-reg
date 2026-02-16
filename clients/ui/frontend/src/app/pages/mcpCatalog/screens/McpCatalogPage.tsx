import * as React from 'react';
import {
  Button,
  EmptyState,
  Gallery,
  PageSection,
  SearchInput,
  Sidebar,
  SidebarContent,
  SidebarPanel,
  Spinner,
  Title,
  Toolbar,
  ToolbarContent,
  ToolbarItem,
} from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';
import { ApplicationsPage } from 'mod-arch-shared';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import {
  MCP_CATALOG_PAGE_TITLE,
  MCP_CATALOG_DESCRIPTION,
} from '~/app/routes/mcpCatalog/mcpCatalog';
import McpCatalogFilters from '../components/McpCatalogFilters';
import McpServerCard from '../components/McpServerCard';

const McpCatalogPage: React.FC = () => {
  const { mcpServersLoaded, mcpServersLoadError, searchTerm, setSearchTerm, filteredServers } =
    React.useContext(McpCatalogContext);

  return (
    <ApplicationsPage
      title={<Title headingLevel="h1">{MCP_CATALOG_PAGE_TITLE}</Title>}
      description={MCP_CATALOG_DESCRIPTION}
      loaded={mcpServersLoaded}
      loadError={mcpServersLoadError}
      errorMessage="Unable to load MCP catalog."
      empty={mcpServersLoaded && filteredServers.length === 0 && !searchTerm}
      provideChildrenPadding
    >
      <Sidebar hasBorder hasGutter>
        <SidebarPanel>
          <McpCatalogFilters />
        </SidebarPanel>
        <SidebarContent>
          <Toolbar>
            <ToolbarContent>
              <ToolbarItem>
                <SearchInput
                  placeholder="Search MCP servers..."
                  value={searchTerm}
                  onChange={(_event, value) => setSearchTerm(value)}
                  onClear={() => setSearchTerm('')}
                  data-testid="mcp-catalog-search"
                />
              </ToolbarItem>
            </ToolbarContent>
          </Toolbar>
          <PageSection isFilled padding={{ default: 'noPadding' }}>
            {!mcpServersLoaded ? (
              <EmptyState>
                <Spinner />
                <Title headingLevel="h4" size="lg">
                  Loading MCP catalog...
                </Title>
              </EmptyState>
            ) : filteredServers.length === 0 ? (
              <EmptyState
                icon={SearchIcon}
                titleText="No MCP servers found"
                headingLevel="h4"
              >
                <p>Adjust your search or filters and try again.</p>
                <Button variant="link" onClick={() => setSearchTerm('')}>
                  Clear search
                </Button>
              </EmptyState>
            ) : (
              <Gallery hasGutter minWidths={{ default: '300px' }}>
                {filteredServers.map((server) => (
                  <McpServerCard key={server.name} server={server} />
                ))}
              </Gallery>
            )}
          </PageSection>
        </SidebarContent>
      </Sidebar>
    </ApplicationsPage>
  );
};

export default McpCatalogPage;

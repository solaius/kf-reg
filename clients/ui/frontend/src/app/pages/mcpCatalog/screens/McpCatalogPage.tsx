import * as React from 'react';
import {
  Button,
  EmptyState,
  Flex,
  Gallery,
  PageSection,
  Sidebar,
  SidebarContent,
  SidebarPanel,
  Spinner,
  Stack,
  StackItem,
  Title,
  ToggleGroup,
  ToggleGroupItem,
  Toolbar,
  ToolbarContent,
  ToolbarFilter,
  ToolbarGroup,
  ToolbarItem,
  ToolbarToggleGroup,
} from '@patternfly/react-core';
import { ArrowRightIcon, FilterIcon, SearchIcon } from '@patternfly/react-icons';
import { useThemeContext } from 'mod-arch-kubeflow';
import { ApplicationsPage } from 'mod-arch-shared';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import {
  MCP_ALL_CATEGORIES,
  MCP_OTHER_SERVERS,
  MCP_OTHER_SERVERS_DISPLAY,
  McpCatalogFilterKey,
} from '~/app/mcpCatalogTypes';
import {
  MCP_CATALOG_PAGE_TITLE,
  MCP_CATALOG_DESCRIPTION,
} from '~/app/routes/mcpCatalog/mcpCatalog';
import ThemeAwareSearchInput from '~/app/pages/modelRegistry/screens/components/ThemeAwareSearchInput';
import McpCatalogFilters from '../components/McpCatalogFilters';
import McpServerCard from '../components/McpServerCard';

const FILTER_CATEGORY_NAMES: Record<McpCatalogFilterKey, string> = {
  [McpCatalogFilterKey.DEPLOYMENT_MODE]: 'Deployment Mode',
  [McpCatalogFilterKey.CATEGORY]: 'Category',
  [McpCatalogFilterKey.LICENSE]: 'License',
  [McpCatalogFilterKey.TRANSPORT]: 'Transport',
};

type SourceLabelBlock = {
  id: string;
  label: string;
  displayName: string;
};

const McpCatalogPage: React.FC = () => {
  const {
    mcpServers,
    mcpServersLoaded,
    mcpServersLoadError,
    searchTerm,
    setSearchTerm,
    filteredServers,
    selectedCategory,
    setSelectedCategory,
    filterData,
    setFilterData,
    clearAllFilters,
    availableFilterValues,
  } = React.useContext(McpCatalogContext);
  const { isMUITheme } = useThemeContext();

  const [inputValue, setInputValue] = React.useState(searchTerm || '');

  React.useEffect(() => {
    setInputValue(searchTerm || '');
  }, [searchTerm]);

  // Build source-label-based tabs like Model Catalog's ModelCatalogSourceLabelBlocks
  const sourceLabelBlocks: SourceLabelBlock[] = React.useMemo(() => {
    const allBlock: SourceLabelBlock = {
      id: 'all',
      label: MCP_ALL_CATEGORIES,
      displayName: MCP_ALL_CATEGORIES,
    };

    const labelBlocks: SourceLabelBlock[] = availableFilterValues.sourceLabels.map((label) => ({
      id: `label-${label}`,
      label,
      displayName: `${label} servers`,
    }));

    const blocks: SourceLabelBlock[] = [allBlock, ...labelBlocks];

    // Check if there are servers without a source label
    const hasServersWithoutLabel = mcpServers.some((s) => !s.sourceLabel);
    if (hasServersWithoutLabel) {
      blocks.push({
        id: 'other',
        label: MCP_OTHER_SERVERS,
        displayName: MCP_OTHER_SERVERS_DISPLAY,
      });
    }

    return blocks;
  }, [availableFilterValues.sourceLabels, mcpServers]);

  const handleModelSearch = () => {
    if (inputValue.trim() !== searchTerm) {
      setSearchTerm(inputValue.trim());
    }
  };

  const handleClear = () => {
    setInputValue('');
    setSearchTerm('');
  };

  const handleSearchInputChange = (value: string) => {
    setInputValue(value);
    setSearchTerm(value.trim());
  };

  const handleSearchInputSearch = (_: React.SyntheticEvent<HTMLButtonElement>, value: string) => {
    setSearchTerm(value.trim());
  };

  const hasActiveFilters =
    Object.values(filterData).some((arr) => arr.length > 0) || Boolean(searchTerm);

  const handleRemoveFilter = (categoryKey: string, labelKey: string) => {
    const key = categoryKey as McpCatalogFilterKey;
    if (filterData[key]) {
      setFilterData(
        key,
        filterData[key].filter((v) => v !== labelKey),
      );
    }
  };

  const handleClearCategory = (categoryKey: string) => {
    const key = categoryKey as McpCatalogFilterKey;
    if (filterData[key]) {
      setFilterData(key, []);
    }
  };

  return (
    <ApplicationsPage
      title={<Title headingLevel="h1">{MCP_CATALOG_PAGE_TITLE}</Title>}
      description={MCP_CATALOG_DESCRIPTION}
      loaded={mcpServersLoaded}
      loadError={mcpServersLoadError}
      errorMessage="Unable to load MCP catalog."
      empty={mcpServersLoaded && filteredServers.length === 0 && !searchTerm && !hasActiveFilters}
      provideChildrenPadding
    >
      <Stack hasGutter>
        {/* Search toolbar */}
        <StackItem>
          <Toolbar
            {...(hasActiveFilters
              ? {
                  clearAllFilters: clearAllFilters,
                  clearFiltersButtonText: 'Reset all filters',
                }
              : {})}
          >
            <ToolbarContent rowWrap={{ default: 'wrap' }}>
              <Flex>
                <ToolbarToggleGroup breakpoint="md" toggleIcon={<FilterIcon />}>
                  <ToolbarGroup variant="filter-group" gap={{ default: 'gapMd' }} alignItems="center">
                    <ToolbarItem>
                      <ThemeAwareSearchInput
                        data-testid="mcp-catalog-search"
                        fieldLabel="Filter by name, description and provider"
                        aria-label="Search with submit button"
                        className="toolbar-fieldset-wrapper"
                        placeholder="Filter by name, description and provider"
                        value={inputValue}
                        style={{
                          minWidth: '600px',
                        }}
                        onChange={handleSearchInputChange}
                        onSearch={handleSearchInputSearch}
                        onClear={handleClear}
                      />
                    </ToolbarItem>
                    <ToolbarItem>
                      {isMUITheme && (
                        <Button
                          isInline
                          aria-label="arrow-right-button"
                          data-testid="mcp-search-button"
                          variant="link"
                          icon={<ArrowRightIcon />}
                          iconPosition="right"
                          onClick={handleModelSearch}
                        />
                      )}
                    </ToolbarItem>
                  </ToolbarGroup>
                </ToolbarToggleGroup>
                {/* Active filter chips */}
                {hasActiveFilters &&
                  Object.entries(filterData).map(([key, values]) => {
                    if (values.length === 0) {
                      return null;
                    }
                    const filterKey = key as McpCatalogFilterKey;
                    const categoryName = FILTER_CATEGORY_NAMES[filterKey];
                    const labels = values.map((value) => ({
                      key: value,
                      node: (
                        <span data-testid={`${filterKey}-filter-chip-${value}`}>{value}</span>
                      ),
                    }));
                    return (
                      <ToolbarFilter
                        key={filterKey}
                        categoryName={{ key: filterKey, name: categoryName }}
                        labels={labels}
                        deleteLabel={(category, label) => {
                          const catKey = typeof category === 'string' ? category : category.key;
                          const labKey = typeof label === 'string' ? label : label.key;
                          handleRemoveFilter(catKey, labKey);
                        }}
                        deleteLabelGroup={(category) => {
                          const catKey = typeof category === 'string' ? category : category.key;
                          handleClearCategory(catKey);
                        }}
                        data-testid={`${filterKey}-filter-container`}
                      >
                        {null}
                      </ToolbarFilter>
                    );
                  })}
              </Flex>
            </ToolbarContent>
          </Toolbar>
        </StackItem>

        {/* Source label tabs */}
        <StackItem>
          <Flex
            justifyContent={{ default: 'justifyContentFlexStart' }}
            alignItems={{ default: 'alignItemsCenter' }}
          >
            <ToggleGroup
              aria-label="Source label selection"
              className="pf-v6-u-pb-md pf-v6-u-pt-md"
            >
              {sourceLabelBlocks.map((block) => (
                <ToggleGroupItem
                  buttonId={block.id}
                  data-testid={block.id}
                  key={block.id}
                  text={block.displayName}
                  isSelected={block.label === selectedCategory}
                  onChange={() => setSelectedCategory(block.label)}
                />
              ))}
            </ToggleGroup>
          </Flex>
        </StackItem>

        {/* Sidebar with filters + gallery */}
        <StackItem isFilled>
          <Sidebar hasBorder hasGutter>
            <SidebarPanel>
              <McpCatalogFilters />
            </SidebarPanel>
            <SidebarContent>
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
                    <Button variant="link" onClick={clearAllFilters}>
                      Clear all filters
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
        </StackItem>
      </Stack>
    </ApplicationsPage>
  );
};

export default McpCatalogPage;

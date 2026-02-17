import * as React from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import {
  Button,
  EmptyState,
  PageSection,
  Spinner,
  Stack,
  StackItem,
  Title,
} from '@patternfly/react-core';
import { SearchIcon } from '@patternfly/react-icons';
import { ApplicationsPage } from 'mod-arch-shared';
import { useCatalogPlugins } from '~/app/context/catalog/CatalogContext';
import { usePluginEntities } from '~/app/hooks/usePluginEntities';
import GenericListView from '~/app/pages/genericCatalog/components/GenericListView';
import GenericFilterBar from '~/app/pages/genericCatalog/components/GenericFilterBar';
import { getFieldValue } from '~/app/pages/genericCatalog/utils';

// Escape single quotes in filter values to prevent filterQuery syntax issues.
// The server-side parser uses parameterized queries, so this is for syntax safety, not SQL injection.
const sanitizeFilterValue = (value: string): string => value.replace(/'/g, "''");

const PluginEntityListPage: React.FC = () => {
  const { pluginName = '', entityPlural = '' } = useParams<{
    pluginName: string;
    entityPlural: string;
  }>();
  const navigate = useNavigate();
  const { getPluginCaps } = useCatalogPlugins();

  const caps = getPluginCaps(pluginName);
  const entityCaps = caps?.entities.find((e) => e.plural === entityPlural);

  const [searchTerm, setSearchTerm] = React.useState('');
  const [activeFilters, setActiveFilters] = React.useState<Record<string, string[]>>({});

  // Build server-side filterQuery from search term and active filters
  const queryParams = React.useMemo(() => {
    const conditions: string[] = [];

    if (searchTerm) {
      conditions.push(`name LIKE '%${sanitizeFilterValue(searchTerm)}%'`);
    }

    if (entityCaps?.fields.filterFields) {
      for (const field of entityCaps.fields.filterFields) {
        const values = activeFilters[field.name];
        if (values && values.length > 0) {
          if (field.type === 'text') {
            conditions.push(`${field.name} LIKE '%${sanitizeFilterValue(values[0])}%'`);
          } else if (values.length === 1) {
            conditions.push(`${field.name}='${sanitizeFilterValue(values[0])}'`);
          } else {
            conditions.push(`${field.name} IN ('${values.map(sanitizeFilterValue).join("','")}')`);
          }
        }
      }
    }

    if (conditions.length === 0) {
      return undefined;
    }
    return { filterQuery: conditions.join(' AND ') };
  }, [searchTerm, activeFilters, entityCaps]);

  const { entities, loaded, error, nextPageToken, totalSize, loadMore, isLoadingMore } =
    usePluginEntities(pluginName, entityPlural, queryParams);

  const handleFilterChange = (field: string, values: string[]) => {
    setActiveFilters((prev) => ({ ...prev, [field]: values }));
  };

  const handleClearAll = () => {
    setActiveFilters({});
    setSearchTerm('');
  };

  // Client-side filtering
  const filteredEntities = React.useMemo(() => {
    let result = entities;

    if (searchTerm && entityCaps) {
      const term = searchTerm.toLowerCase();
      const nameField = entityCaps.uiHints?.nameField || 'name';
      result = result.filter((item) => {
        const name = String(getFieldValue(item, nameField) || '').toLowerCase();
        const desc = String(getFieldValue(item, 'description') || '').toLowerCase();
        return name.includes(term) || desc.includes(term);
      });
    }

    // Apply filter fields
    if (entityCaps?.fields.filterFields) {
      for (const field of entityCaps.fields.filterFields) {
        const values = activeFilters[field.name];
        if (values && values.length > 0) {
          if (field.type === 'boolean') {
            result = result.filter((item) => {
              const val = getFieldValue(item, field.name);
              return values.includes('true') ? Boolean(val) : !val;
            });
          } else {
            result = result.filter((item) => {
              const val = String(getFieldValue(item, field.name) || '');
              return values.includes(val);
            });
          }
        }
      }
    }

    return result;
  }, [entities, searchTerm, activeFilters, entityCaps]);

  const handleEntityClick = (name: string) => {
    navigate(`/catalog/${pluginName}/${entityPlural}/${name}`);
  };

  const displayName = entityCaps?.displayName || entityPlural;
  const pluginDisplayName = caps?.plugin.displayName || caps?.plugin.name || pluginName;

  return (
    <ApplicationsPage
      title={<Title headingLevel="h1">{displayName}</Title>}
      description={entityCaps?.description || `Browse ${displayName} from ${pluginDisplayName}`}
      loaded={loaded}
      loadError={error}
      errorMessage={`Unable to load ${displayName}.`}
      empty={loaded && entities.length === 0}
      provideChildrenPadding
    >
      <Stack hasGutter>
        {entityCaps?.fields.filterFields && entityCaps.fields.filterFields.length > 0 && (
          <StackItem>
            <GenericFilterBar
              filterFields={entityCaps.fields.filterFields}
              activeFilters={activeFilters}
              searchTerm={searchTerm}
              onSearchChange={setSearchTerm}
              onFilterChange={handleFilterChange}
              onClearAll={handleClearAll}
            />
          </StackItem>
        )}
        {!entityCaps?.fields.filterFields && (
          <StackItem>
            <GenericFilterBar
              filterFields={[]}
              activeFilters={{}}
              searchTerm={searchTerm}
              onSearchChange={setSearchTerm}
              onFilterChange={handleFilterChange}
              onClearAll={handleClearAll}
            />
          </StackItem>
        )}
        <StackItem isFilled>
          <PageSection isFilled padding={{ default: 'noPadding' }}>
            {!loaded ? (
              <EmptyState>
                <Spinner />
                <Title headingLevel="h4" size="lg">
                  Loading {displayName}...
                </Title>
              </EmptyState>
            ) : filteredEntities.length === 0 ? (
              <EmptyState icon={SearchIcon} titleText={`No ${displayName} found`} headingLevel="h4">
                <p>Adjust your search or filters and try again.</p>
              </EmptyState>
            ) : entityCaps ? (
              <GenericListView
                entity={entityCaps}
                entities={filteredEntities}
                onEntityClick={handleEntityClick}
              />
            ) : (
              <EmptyState titleText="Unknown entity type" headingLevel="h4">
                <p>Capabilities for this entity type are not yet loaded.</p>
              </EmptyState>
            )}
          </PageSection>
        </StackItem>
        {nextPageToken && (
          <StackItem>
            <Button
              variant="secondary"
              onClick={loadMore}
              isLoading={isLoadingMore}
              isDisabled={isLoadingMore}
            >
              Load more
            </Button>
            {totalSize > 0 && (
              <span style={{ marginLeft: '1rem' }}>
                Showing {filteredEntities.length} of {totalSize}
              </span>
            )}
          </StackItem>
        )}
      </Stack>
    </ApplicationsPage>
  );
};

export default PluginEntityListPage;

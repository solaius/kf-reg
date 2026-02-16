import React from 'react';
import {
  Button,
  Checkbox,
  Content,
  ContentVariants,
  Divider,
  SearchInput,
} from '@patternfly/react-core';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import { McpCatalogFilterKey } from '~/app/mcpCatalogTypes';

const MAX_VISIBLE_FILTERS = 5;

type McpFilterSectionProps = {
  title: string;
  filterKey: McpCatalogFilterKey;
  values: string[];
  selectedValues: string[];
  onFilterChange: (key: McpCatalogFilterKey, value: string, checked: boolean) => void;
};

const McpFilterSection: React.FC<McpFilterSectionProps> = ({
  title,
  filterKey,
  values,
  selectedValues,
  onFilterChange,
}) => {
  const [showMore, setShowMore] = React.useState(false);
  const [searchValue, setSearchValue] = React.useState('');

  const filteredValues = React.useMemo(
    () =>
      values.filter(
        (v) =>
          v.toLowerCase().includes(searchValue.trim().toLowerCase()) ||
          selectedValues.includes(v),
      ),
    [values, searchValue, selectedValues],
  );

  const visibleValues = showMore ? filteredValues : filteredValues.slice(0, MAX_VISIBLE_FILTERS);

  return (
    <Content data-testid={`${title}-filter`}>
      <Content component={ContentVariants.h6}>{title}</Content>
      {values.length > MAX_VISIBLE_FILTERS && (
        <SearchInput
          placeholder={`Search ${title.toLowerCase()}`}
          data-testid={`${title}-filter-search`}
          className="pf-v6-u-mb-sm"
          value={searchValue}
          onChange={(_event, newValue) => setSearchValue(newValue)}
        />
      )}
      {visibleValues.length === 0 && (
        <div data-testid={`${title}-filter-empty`}>No results found</div>
      )}
      {visibleValues.map((value) => (
        <Checkbox
          key={value}
          id={`filter-${filterKey}-${value}`}
          label={value}
          isChecked={selectedValues.includes(value)}
          onChange={(_, checked) => onFilterChange(filterKey, value, checked)}
          data-testid={`${title}-${value}-checkbox`}
        />
      ))}
      {!showMore && filteredValues.length > MAX_VISIBLE_FILTERS && (
        <Button
          variant="link"
          onClick={() => setShowMore(true)}
          data-testid={`${title}-filter-show-more`}
        >
          Show more
        </Button>
      )}
      {showMore && filteredValues.length > MAX_VISIBLE_FILTERS && (
        <Button
          variant="link"
          onClick={() => setShowMore(false)}
          data-testid={`${title}-filter-show-less`}
        >
          Show less
        </Button>
      )}
    </Content>
  );
};

const McpCatalogFilters: React.FC = () => {
  const { filterData, setFilterData, clearAllFilters, availableFilterValues } =
    React.useContext(McpCatalogContext);

  const handleFilterChange = (key: McpCatalogFilterKey, value: string, checked: boolean) => {
    const current = filterData[key];
    setFilterData(key, checked ? [...current, value] : current.filter((v) => v !== value));
  };

  const hasFilters = Object.values(filterData).some((arr) => arr.length > 0);

  return (
    <div>
      <Content component={ContentVariants.h5} className="pf-v6-u-mb-md">
        Filters
        {hasFilters && (
          <Button variant="link" onClick={clearAllFilters} className="pf-v6-u-ml-md">
            Clear all
          </Button>
        )}
      </Content>
      <McpFilterSection
        title="Deployment Mode"
        filterKey={McpCatalogFilterKey.DEPLOYMENT_MODE}
        values={availableFilterValues.deploymentModes}
        selectedValues={filterData[McpCatalogFilterKey.DEPLOYMENT_MODE]}
        onFilterChange={handleFilterChange}
      />
      <Divider className="pf-v6-u-my-md" />
      <McpFilterSection
        title="Category"
        filterKey={McpCatalogFilterKey.CATEGORY}
        values={availableFilterValues.categories}
        selectedValues={filterData[McpCatalogFilterKey.CATEGORY]}
        onFilterChange={handleFilterChange}
      />
      <Divider className="pf-v6-u-my-md" />
      <McpFilterSection
        title="License"
        filterKey={McpCatalogFilterKey.LICENSE}
        values={availableFilterValues.licenses}
        selectedValues={filterData[McpCatalogFilterKey.LICENSE]}
        onFilterChange={handleFilterChange}
      />
      <Divider className="pf-v6-u-my-md" />
      <McpFilterSection
        title="Transport"
        filterKey={McpCatalogFilterKey.TRANSPORT}
        values={availableFilterValues.transports}
        selectedValues={filterData[McpCatalogFilterKey.TRANSPORT]}
        onFilterChange={handleFilterChange}
      />
    </div>
  );
};

export default McpCatalogFilters;

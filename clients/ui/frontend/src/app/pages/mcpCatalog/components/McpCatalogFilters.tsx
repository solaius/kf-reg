import React from 'react';
import {
  Button,
  Checkbox,
  Form,
  FormGroup,
  FormSection,
  Title,
} from '@patternfly/react-core';
import { McpCatalogContext } from '~/app/context/mcpCatalog/McpCatalogContext';
import { McpCatalogFilterKey } from '~/app/mcpCatalogTypes';

const McpCatalogFilters: React.FC = () => {
  const { filterData, setFilterData } = React.useContext(McpCatalogContext);

  const handleFilterChange = (key: McpCatalogFilterKey, value: string, checked: boolean) => {
    const current = filterData[key];
    if (checked) {
      setFilterData(key, [...current, value]);
    } else {
      setFilterData(key, current.filter((v) => v !== value));
    }
  };

  const clearAllFilters = () => {
    setFilterData(McpCatalogFilterKey.DEPLOYMENT_MODE, []);
    setFilterData(McpCatalogFilterKey.CATEGORY, []);
    setFilterData(McpCatalogFilterKey.LICENSE, []);
    setFilterData(McpCatalogFilterKey.TRANSPORT, []);
  };

  const hasFilters =
    filterData[McpCatalogFilterKey.DEPLOYMENT_MODE].length > 0 ||
    filterData[McpCatalogFilterKey.CATEGORY].length > 0 ||
    filterData[McpCatalogFilterKey.LICENSE].length > 0 ||
    filterData[McpCatalogFilterKey.TRANSPORT].length > 0;

  return (
    <Form>
      <Title headingLevel="h3" size="md" className="pf-v6-u-mb-md">
        Filters
        {hasFilters && (
          <Button variant="link" onClick={clearAllFilters} className="pf-v6-u-ml-md">
            Clear all
          </Button>
        )}
      </Title>

      <FormSection title="Deployment Mode">
        <FormGroup>
          {['local', 'remote'].map((mode) => (
            <Checkbox
              key={mode}
              id={`filter-deployment-${mode}`}
              label={mode.charAt(0).toUpperCase() + mode.slice(1)}
              isChecked={filterData[McpCatalogFilterKey.DEPLOYMENT_MODE].includes(mode)}
              onChange={(_event, checked) =>
                handleFilterChange(McpCatalogFilterKey.DEPLOYMENT_MODE, mode, checked)
              }
            />
          ))}
        </FormGroup>
      </FormSection>

      <FormSection title="Category">
        <FormGroup>
          {['Red Hat', 'DevOps', 'Database', 'Communication'].map((cat) => (
            <Checkbox
              key={cat}
              id={`filter-category-${cat}`}
              label={cat}
              isChecked={filterData[McpCatalogFilterKey.CATEGORY].includes(cat)}
              onChange={(_event, checked) =>
                handleFilterChange(McpCatalogFilterKey.CATEGORY, cat, checked)
              }
            />
          ))}
        </FormGroup>
      </FormSection>

      <FormSection title="License">
        <FormGroup>
          {['Apache-2.0', 'MIT', 'PostgreSQL'].map((lic) => (
            <Checkbox
              key={lic}
              id={`filter-license-${lic}`}
              label={lic}
              isChecked={filterData[McpCatalogFilterKey.LICENSE].includes(lic)}
              onChange={(_event, checked) =>
                handleFilterChange(McpCatalogFilterKey.LICENSE, lic, checked)
              }
            />
          ))}
        </FormGroup>
      </FormSection>

      <FormSection title="Transport">
        <FormGroup>
          {['stdio', 'http', 'sse'].map((transport) => (
            <Checkbox
              key={transport}
              id={`filter-transport-${transport}`}
              label={transport.toUpperCase()}
              isChecked={filterData[McpCatalogFilterKey.TRANSPORT].includes(transport)}
              onChange={(_event, checked) =>
                handleFilterChange(McpCatalogFilterKey.TRANSPORT, transport, checked)
              }
            />
          ))}
        </FormGroup>
      </FormSection>
    </Form>
  );
};

export default McpCatalogFilters;

import * as React from 'react';
import {
  SearchInput,
  Select,
  SelectOption,
  MenuToggle,
  MenuToggleElement,
  Checkbox,
  Toolbar,
  ToolbarContent,
  ToolbarItem,
  ToolbarFilter,
  ToolbarToggleGroup,
  ToolbarLabelGroup,
  Button,
} from '@patternfly/react-core';
import { FilterIcon } from '@patternfly/react-icons';
import { V2FilterField } from '~/app/types/capabilities';

type GenericFilterBarProps = {
  filterFields: V2FilterField[];
  activeFilters: Record<string, string[]>;
  searchTerm: string;
  onSearchChange: (value: string) => void;
  onFilterChange: (field: string, values: string[]) => void;
  onClearAll: () => void;
};

const GenericFilterBar: React.FC<GenericFilterBarProps> = ({
  filterFields,
  activeFilters,
  searchTerm,
  onSearchChange,
  onFilterChange,
  onClearAll,
}) => {
  const [openSelects, setOpenSelects] = React.useState<Record<string, boolean>>({});

  const hasActiveFilters =
    Object.values(activeFilters).some((arr) => arr.length > 0) || Boolean(searchTerm);

  const toggleSelect = (field: string) => {
    setOpenSelects((prev) => ({ ...prev, [field]: !prev[field] }));
  };

  const handleSelectChange = (field: string, value: string) => {
    const current = activeFilters[field] || [];
    const next = current.includes(value)
      ? current.filter((v) => v !== value)
      : [...current, value];
    onFilterChange(field, next);
  };

  return (
    <Toolbar
      {...(hasActiveFilters
        ? {
            clearAllFilters: onClearAll,
            clearFiltersButtonText: 'Clear all filters',
          }
        : {})}
    >
      <ToolbarContent>
        <ToolbarToggleGroup breakpoint="md" toggleIcon={<FilterIcon />}>
          <ToolbarItem>
            <SearchInput
              placeholder="Search..."
              value={searchTerm}
              onChange={(_e, value) => onSearchChange(value)}
              onClear={() => onSearchChange('')}
              aria-label="Search entities"
            />
          </ToolbarItem>
          {filterFields.map((field) => {
            if (field.type === 'text') {
              return (
                <ToolbarItem key={field.name}>
                  <SearchInput
                    placeholder={field.displayName}
                    value={(activeFilters[field.name] || [])[0] || ''}
                    onChange={(_e, value) => onFilterChange(field.name, value ? [value] : [])}
                    onClear={() => onFilterChange(field.name, [])}
                    aria-label={field.displayName}
                  />
                </ToolbarItem>
              );
            }
            if (field.type === 'boolean') {
              const isChecked = (activeFilters[field.name] || []).includes('true');
              return (
                <ToolbarItem key={field.name}>
                  <Checkbox
                    id={`filter-${field.name}`}
                    label={field.displayName}
                    isChecked={isChecked}
                    onChange={(_e, checked) =>
                      onFilterChange(field.name, checked ? ['true'] : [])
                    }
                  />
                </ToolbarItem>
              );
            }
            if (field.type === 'select' && field.options) {
              const currentValues = activeFilters[field.name] || [];
              const categoryLabelGroup: ToolbarLabelGroup = {
                key: field.name,
                name: field.displayName,
              };
              return (
                <ToolbarFilter
                  key={field.name}
                  categoryName={categoryLabelGroup}
                  labels={currentValues}
                  deleteLabel={(_category, label) => {
                    const labelStr = typeof label === 'string' ? label : label.key;
                    handleSelectChange(field.name, labelStr);
                  }}
                  deleteLabelGroup={() => onFilterChange(field.name, [])}
                >
                  <Select
                    isOpen={openSelects[field.name] || false}
                    onOpenChange={() => toggleSelect(field.name)}
                    onSelect={(_e, value) => {
                      if (typeof value === 'string') {
                        handleSelectChange(field.name, value);
                      }
                    }}
                    toggle={(toggleRef: React.Ref<MenuToggleElement>) => (
                      <MenuToggle
                        ref={toggleRef}
                        onClick={() => toggleSelect(field.name)}
                        isExpanded={openSelects[field.name] || false}
                      >
                        {field.displayName}
                        {currentValues.length > 0 && ` (${currentValues.length})`}
                      </MenuToggle>
                    )}
                  >
                    {(field.options || []).map((opt) => (
                      <SelectOption
                        key={opt}
                        value={opt}
                        hasCheckbox
                        isSelected={currentValues.includes(opt)}
                      >
                        {opt}
                      </SelectOption>
                    ))}
                  </Select>
                </ToolbarFilter>
              );
            }
            return null;
          })}
        </ToolbarToggleGroup>
        {hasActiveFilters && (
          <ToolbarItem>
            <Button variant="link" onClick={onClearAll}>
              Clear all filters
            </Button>
          </ToolbarItem>
        )}
      </ToolbarContent>
    </Toolbar>
  );
};

export default GenericFilterBar;

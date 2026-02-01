# Component Library

This document covers the UI component libraries used in the frontend.

## Overview

The frontend uses two component libraries:

| Library | Version | Usage |
|---------|---------|-------|
| **PatternFly** | 6.4.0 | Primary component library |
| **Material-UI** | 7.3.4 | Additional components, theme-conditional |

## PatternFly Usage

PatternFly is the primary component library, providing the Red Hat design system.

### Core Components

```typescript
import {
  // Layout
  Page, PageSection, PageSidebar, PageHeader,
  Masthead, MastheadMain, MastheadBrand, MastheadContent,
  Sidebar, SidebarPanel, SidebarContent,

  // Navigation
  Nav, NavItem, NavList, NavGroup,
  Breadcrumb, BreadcrumbItem,

  // Content
  Card, CardHeader, CardTitle, CardBody, CardFooter,
  Gallery, GalleryItem,
  Grid, GridItem,
  Flex, FlexItem,

  // Forms
  Button, TextInput, Select, SelectOption,
  Checkbox, Radio, Switch,
  Form, FormGroup, FormSection,

  // Feedback
  Alert, AlertGroup,
  Modal, ModalVariant,
  Spinner,
  EmptyState, EmptyStateBody, EmptyStateIcon,

  // Data Display
  Label, LabelGroup,
  Badge,
  Tooltip,
  Popover,

  // Toolbar
  Toolbar, ToolbarContent, ToolbarItem, ToolbarGroup,
  SearchInput,
} from '@patternfly/react-core';
```

### Table Components

```typescript
import {
  Table, Thead, Tbody, Tr, Th, Td,
  TableVariant, TableComposable,
  ActionsColumn, IAction,
  SortColumn,
} from '@patternfly/react-table';
```

### Icons

```typescript
import {
  CheckCircleIcon,
  ExclamationTriangleIcon,
  InfoCircleIcon,
  TimesCircleIcon,
  CubeIcon,
  CogIcon,
  PlusIcon,
  TrashIcon,
  PencilAltIcon,
  SearchIcon,
  FilterIcon,
} from '@patternfly/react-icons';
```

### Template Components

```typescript
import {
  SimpleSelect,
  MultipleSelect,
} from '@patternfly/react-templates';
```

### Example: Page Layout

```typescript
const MyPage: React.FC = () => (
  <Page sidebar={<AppNavSidebar />}>
    <PageSection variant="light">
      <Breadcrumb>
        <BreadcrumbItem>Home</BreadcrumbItem>
        <BreadcrumbItem isActive>Models</BreadcrumbItem>
      </Breadcrumb>
    </PageSection>

    <PageSection>
      <Card>
        <CardHeader>
          <CardTitle>Model List</CardTitle>
        </CardHeader>
        <CardBody>
          <Gallery hasGutter>
            {models.map(model => (
              <GalleryItem key={model.id}>
                <ModelCard model={model} />
              </GalleryItem>
            ))}
          </Gallery>
        </CardBody>
      </Card>
    </PageSection>
  </Page>
);
```

### Example: Toolbar with Filters

```typescript
const ModelToolbar: React.FC = () => (
  <Toolbar>
    <ToolbarContent>
      <ToolbarGroup variant="filter-group">
        <ToolbarItem>
          <SearchInput
            placeholder="Search models..."
            value={searchTerm}
            onChange={(_, value) => setSearchTerm(value)}
            onClear={() => setSearchTerm('')}
          />
        </ToolbarItem>
        <ToolbarItem>
          <SimpleSelect
            initialOptions={providerOptions}
            value={selectedProvider}
            onChange={(value) => setSelectedProvider(value)}
            placeholder="Provider"
          />
        </ToolbarItem>
      </ToolbarGroup>
      <ToolbarGroup variant="action-group-plain">
        <ToolbarItem>
          <Button variant="primary">
            <PlusIcon /> Add Model
          </Button>
        </ToolbarItem>
      </ToolbarGroup>
    </ToolbarContent>
  </Toolbar>
);
```

### Example: Data Table

```typescript
const ModelTable: React.FC<{ models: Model[] }> = ({ models }) => (
  <Table aria-label="Models table" variant={TableVariant.compact}>
    <Thead>
      <Tr>
        <Th>Name</Th>
        <Th>Provider</Th>
        <Th>Status</Th>
        <Th />
      </Tr>
    </Thead>
    <Tbody>
      {models.map(model => (
        <Tr key={model.id}>
          <Td dataLabel="Name">{model.name}</Td>
          <Td dataLabel="Provider">{model.provider}</Td>
          <Td dataLabel="Status">
            <Label color={model.status === 'active' ? 'green' : 'orange'}>
              {model.status}
            </Label>
          </Td>
          <Td isActionCell>
            <ActionsColumn
              items={[
                { title: 'Edit', onClick: () => onEdit(model) },
                { title: 'Delete', onClick: () => onDelete(model) },
              ]}
            />
          </Td>
        </Tr>
      ))}
    </Tbody>
  </Table>
);
```

### Example: Empty State

```typescript
const EmptyModelList: React.FC = () => (
  <EmptyState variant={EmptyStateVariant.lg}>
    <EmptyStateIcon icon={CubeIcon} />
    <Title headingLevel="h4" size="lg">
      No models found
    </Title>
    <EmptyStateBody>
      There are no models matching your criteria. Try adjusting your filters
      or create a new model.
    </EmptyStateBody>
    <Button variant="primary">Create Model</Button>
  </EmptyState>
);
```

## Material-UI Usage

Material-UI is used conditionally based on theme context.

### Theme-Aware Rendering

```typescript
import { useThemeContext } from 'mod-arch-kubeflow';

const MyComponent: React.FC = () => {
  const { theme } = useThemeContext();
  const isMuiTheme = theme === 'mui';

  if (isMuiTheme) {
    return <MuiButton variant="contained">Click Me</MuiButton>;
  }

  return <Button variant="primary">Click Me</Button>;
};
```

### Common MUI Components

```typescript
import {
  // Core
  Box, Container, Paper,
  Typography,
  Button, IconButton,

  // Forms
  TextField, Select, MenuItem,
  Checkbox, Radio, Switch,

  // Feedback
  Alert, Snackbar,
  CircularProgress,
  Dialog, DialogTitle, DialogContent, DialogActions,

  // Data Display
  Chip, Badge,
  Tooltip,
  Table, TableHead, TableBody, TableRow, TableCell,

  // Layout
  Grid, Stack,
  Divider,
} from '@mui/material';

import {
  Add as AddIcon,
  Delete as DeleteIcon,
  Edit as EditIcon,
  Search as SearchIcon,
} from '@mui/icons-material';
```

## Shared Components

Custom reusable components built on top of PatternFly/MUI.

### ConfirmModal

```typescript
// app/shared/components/ConfirmModal.tsx
interface ConfirmModalProps {
  isOpen: boolean;
  title: string;
  message: string;
  confirmText?: string;
  cancelText?: string;
  variant?: 'warning' | 'danger';
  onConfirm: () => void;
  onCancel: () => void;
}

const ConfirmModal: React.FC<ConfirmModalProps> = ({
  isOpen,
  title,
  message,
  confirmText = 'Confirm',
  cancelText = 'Cancel',
  variant = 'warning',
  onConfirm,
  onCancel,
}) => (
  <Modal
    variant={ModalVariant.small}
    title={title}
    isOpen={isOpen}
    onClose={onCancel}
    actions={[
      <Button key="confirm" variant={variant === 'danger' ? 'danger' : 'primary'} onClick={onConfirm}>
        {confirmText}
      </Button>,
      <Button key="cancel" variant="link" onClick={onCancel}>
        {cancelText}
      </Button>,
    ]}
  >
    {message}
  </Modal>
);
```

### LoadingSpinner

```typescript
// app/shared/components/LoadingSpinner.tsx
interface LoadingSpinnerProps {
  size?: 'sm' | 'md' | 'lg' | 'xl';
  message?: string;
}

const LoadingSpinner: React.FC<LoadingSpinnerProps> = ({
  size = 'lg',
  message = 'Loading...',
}) => (
  <Bullseye>
    <Flex direction={{ default: 'column' }} alignItems={{ default: 'alignItemsCenter' }}>
      <FlexItem>
        <Spinner size={size} />
      </FlexItem>
      {message && (
        <FlexItem>
          <Text component="small">{message}</Text>
        </FlexItem>
      )}
    </Flex>
  </Bullseye>
);
```

### SearchInput with Debounce

```typescript
// app/shared/components/DebouncedSearchInput.tsx
interface DebouncedSearchInputProps {
  value: string;
  onChange: (value: string) => void;
  placeholder?: string;
  debounceMs?: number;
}

const DebouncedSearchInput: React.FC<DebouncedSearchInputProps> = ({
  value,
  onChange,
  placeholder = 'Search...',
  debounceMs = 300,
}) => {
  const [localValue, setLocalValue] = useState(value);
  const debouncedOnChange = useMemo(
    () => debounce(onChange, debounceMs),
    [onChange, debounceMs]
  );

  useEffect(() => {
    setLocalValue(value);
  }, [value]);

  const handleChange = (_: React.SyntheticEvent, newValue: string) => {
    setLocalValue(newValue);
    debouncedOnChange(newValue);
  };

  return (
    <SearchInput
      placeholder={placeholder}
      value={localValue}
      onChange={handleChange}
      onClear={() => {
        setLocalValue('');
        onChange('');
      }}
    />
  );
};
```

## Domain-Specific Components

### Model Catalog Card

```typescript
// app/pages/modelCatalog/components/ModelCatalogCard.tsx
interface ModelCatalogCardProps {
  model: CatalogModel;
  onClick?: () => void;
}

const ModelCatalogCard: React.FC<ModelCatalogCardProps> = ({ model, onClick }) => (
  <Card isClickable isSelectable onClick={onClick}>
    <CardHeader>
      <Flex>
        <FlexItem>
          {model.logo && <img src={model.logo} alt="" width={48} height={48} />}
        </FlexItem>
        <FlexItem flex={{ default: 'flex_1' }}>
          <CardTitle>{model.name}</CardTitle>
          <Text component="small">{model.provider}</Text>
        </FlexItem>
      </Flex>
    </CardHeader>
    <CardBody>
      <p>{model.description}</p>
      <LabelGroup>
        {model.tasks?.slice(0, 3).map(task => (
          <Label key={task} color="blue">{task}</Label>
        ))}
      </LabelGroup>
    </CardBody>
    <CardFooter>
      <Flex>
        {model.license && <Label color="grey">{model.license}</Label>}
      </Flex>
    </CardFooter>
  </Card>
);
```

### MCP Security Indicators

```typescript
// app/pages/mcpCatalog/components/McpSecurityIndicators.tsx
interface McpSecurityIndicatorsProps {
  indicators: McpSecurityIndicator;
}

const indicatorConfig = [
  { key: 'verifiedSource', icon: CheckCircleIcon, label: 'Verified Source', color: 'green' },
  { key: 'secureEndpoint', icon: LockIcon, label: 'Secure Endpoint', color: 'blue' },
  { key: 'sast', icon: ShieldAltIcon, label: 'SAST Scanned', color: 'purple' },
  { key: 'readOnlyTools', icon: EyeIcon, label: 'Read-Only Tools', color: 'cyan' },
];

const McpSecurityIndicators: React.FC<McpSecurityIndicatorsProps> = ({ indicators }) => (
  <Flex gap={{ default: 'gapSm' }}>
    {indicatorConfig.map(({ key, icon: Icon, label, color }) => (
      indicators[key as keyof McpSecurityIndicator] && (
        <Tooltip key={key} content={label}>
          <Label color={color as LabelColor} icon={<Icon />}>
            {label}
          </Label>
        </Tooltip>
      )
    ))}
  </Flex>
);
```

## CSS and Styling

### PatternFly CSS

```typescript
// Import in bootstrap.tsx or index.ts
import '@patternfly/react-core/dist/styles/base.css';
import '@patternfly/react-styles/css/components/Table/table.css';
```

### Custom Styles

```scss
// Custom CSS using PatternFly variables
.my-custom-card {
  --pf-v5-c-card--BackgroundColor: var(--pf-v5-global--palette--blue-50);
  border-left: 4px solid var(--pf-v5-global--primary-color--100);
}

.my-highlight {
  color: var(--pf-v5-global--success-color--100);
}
```

### CSS Modules

```typescript
// Component.module.css
.container {
  padding: var(--pf-v5-global--spacer--md);
}

// Component.tsx
import styles from './Component.module.css';

const Component = () => (
  <div className={styles.container}>...</div>
);
```

---

[Back to Frontend Index](./README.md) | [Previous: State Management](./state-management.md) | [Next: Routing](./routing.md)

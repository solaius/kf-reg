# UI Design Requirements

This document outlines the UI/UX patterns and component guidelines for the Kubeflow Model Registry frontend.

## Design System

### PatternFly

The primary UI framework is **PatternFly 6.4**, RedHat's open source design system. PatternFly provides:

- Consistent component library
- Accessibility compliance (WCAG 2.1)
- Enterprise-ready patterns
- React component bindings

**Documentation**: [PatternFly](https://www.patternfly.org/)

### Material UI (MUI)

**MUI 7.3** is used for specialized components:

- Data grids
- Date/time pickers
- Charts and visualizations

**Documentation**: [MUI](https://mui.com/)

## Component Hierarchy

### Layout Components

```
┌─────────────────────────────────────────────────────────────────┐
│  Page (Full layout with nav)                                    │
│  ├─ Masthead (Header with navigation)                          │
│  ├─ Sidebar (Navigation menu)                                   │
│  └─ PageContent                                                  │
│      ├─ PageSection (Main content sections)                     │
│      ├─ Toolbar (Actions and filters)                           │
│      └─ Content (Data display)                                  │
└─────────────────────────────────────────────────────────────────┘
```

### Content Components

```
┌─────────────────────────────────────────────────────────────────┐
│  Content Patterns                                                │
│                                                                   │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │   Card View     │  │   Table View    │  │  Gallery View   │ │
│  │   (Overview)    │  │   (Lists)       │  │   (Catalog)     │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│                                                                   │
│  ┌─────────────────┐  ┌─────────────────┐  ┌─────────────────┐ │
│  │  Detail View    │  │   Modal/Drawer  │  │   Empty State   │ │
│  │  (Single item)  │  │   (Actions)     │  │   (No data)     │ │
│  └─────────────────┘  └─────────────────┘  └─────────────────┘ │
│                                                                   │
└─────────────────────────────────────────────────────────────────┘
```

## Page Structure

### Standard Page Layout

```tsx
import {
  Page,
  PageSection,
  PageSectionVariants,
  Title,
  Toolbar,
} from '@patternfly/react-core';

const ModelRegistryPage: React.FC = () => (
  <Page>
    <PageSection variant={PageSectionVariants.light}>
      <Title headingLevel="h1">Registered Models</Title>
    </PageSection>
    <PageSection variant={PageSectionVariants.light} padding={{ default: 'noPadding' }}>
      <Toolbar>
        {/* Filters, search, actions */}
      </Toolbar>
    </PageSection>
    <PageSection isFilled variant={PageSectionVariants.default}>
      {/* Main content */}
    </PageSection>
  </Page>
);
```

### Page Sections

| Variant | Use Case |
|---------|----------|
| `light` | Headers, titles, toolbars |
| `default` | Main content area |
| `darker` | Contrast sections |

## Common Patterns

### Table with Toolbar

```tsx
<Toolbar>
  <ToolbarContent>
    <ToolbarItem>
      <SearchInput placeholder="Search by name" />
    </ToolbarItem>
    <ToolbarItem>
      <FilterSelect options={filterOptions} />
    </ToolbarItem>
    <ToolbarItem variant="separator" />
    <ToolbarItem>
      <Button variant="primary">Register Model</Button>
    </ToolbarItem>
  </ToolbarContent>
</Toolbar>

<Table aria-label="Registered models">
  <Thead>
    <Tr>
      <Th>Name</Th>
      <Th>Description</Th>
      <Th>State</Th>
      <Th>Actions</Th>
    </Tr>
  </Thead>
  <Tbody>
    {models.map((model) => (
      <Tr key={model.id}>
        <Td>{model.name}</Td>
        <Td>{model.description}</Td>
        <Td><Label color={getStateColor(model.state)}>{model.state}</Label></Td>
        <Td><ActionsColumn items={getActions(model)} /></Td>
      </Tr>
    ))}
  </Tbody>
</Table>
```

### Empty State

```tsx
<EmptyState>
  <EmptyStateIcon icon={CubesIcon} />
  <Title headingLevel="h2" size="lg">
    No registered models
  </Title>
  <EmptyStateBody>
    Register your first model to get started.
  </EmptyStateBody>
  <Button variant="primary">Register Model</Button>
</EmptyState>
```

### Loading State

```tsx
<Bullseye>
  <Spinner size="xl" />
</Bullseye>
```

### Error State

```tsx
<Alert variant="danger" title="Error loading models">
  <p>{error.message}</p>
  <Button variant="link" onClick={retry}>Retry</Button>
</Alert>
```

## Form Patterns

### Standard Form Layout

```tsx
<Form>
  <FormGroup
    label="Model Name"
    isRequired
    fieldId="model-name"
    helperText="A unique name for your model"
    helperTextInvalid={errors.name}
    validated={errors.name ? 'error' : 'default'}
  >
    <TextInput
      id="model-name"
      value={name}
      onChange={(_, value) => setName(value)}
      isRequired
    />
  </FormGroup>

  <FormGroup label="Description" fieldId="description">
    <TextArea
      id="description"
      value={description}
      onChange={(_, value) => setDescription(value)}
      rows={4}
    />
  </FormGroup>

  <ActionGroup>
    <Button variant="primary" type="submit">Save</Button>
    <Button variant="link" onClick={onCancel}>Cancel</Button>
  </ActionGroup>
</Form>
```

### Form Validation

```tsx
const validateForm = (): boolean => {
  const newErrors: FormErrors = {};

  if (!name.trim()) {
    newErrors.name = 'Name is required';
  }

  if (name.length > 100) {
    newErrors.name = 'Name must be 100 characters or less';
  }

  setErrors(newErrors);
  return Object.keys(newErrors).length === 0;
};
```

## Modal Patterns

### Confirmation Modal

```tsx
<Modal
  variant={ModalVariant.small}
  title="Delete Model?"
  isOpen={isOpen}
  onClose={onClose}
  actions={[
    <Button key="delete" variant="danger" onClick={onDelete}>
      Delete
    </Button>,
    <Button key="cancel" variant="link" onClick={onClose}>
      Cancel
    </Button>,
  ]}
>
  <p>Are you sure you want to delete "{model.name}"?</p>
  <p>This action cannot be undone.</p>
</Modal>
```

### Form Modal

```tsx
<Modal
  variant={ModalVariant.medium}
  title="Register Model"
  isOpen={isOpen}
  onClose={onClose}
>
  <Form>
    {/* Form fields */}
  </Form>
</Modal>
```

## Navigation

### Breadcrumbs

```tsx
<Breadcrumb>
  <BreadcrumbItem>
    <Link to="/model-registry">Model Registry</Link>
  </BreadcrumbItem>
  <BreadcrumbItem>
    <Link to={`/model-registry/${registryId}`}>{registry.name}</Link>
  </BreadcrumbItem>
  <BreadcrumbItem isActive>
    {model.name}
  </BreadcrumbItem>
</Breadcrumb>
```

### Tabs

```tsx
<Tabs activeKey={activeTab} onSelect={(_, key) => setActiveTab(key)}>
  <Tab eventKey="details" title={<TabTitleText>Details</TabTitleText>}>
    <ModelDetails model={model} />
  </Tab>
  <Tab eventKey="versions" title={<TabTitleText>Versions</TabTitleText>}>
    <ModelVersionList modelId={model.id} />
  </Tab>
  <Tab eventKey="artifacts" title={<TabTitleText>Artifacts</TabTitleText>}>
    <ArtifactList modelId={model.id} />
  </Tab>
</Tabs>
```

## Accessibility

### Required Practices

1. **Keyboard Navigation**: All interactive elements must be keyboard accessible
2. **ARIA Labels**: Use `aria-label` for icon-only buttons
3. **Focus Management**: Manage focus when modals open/close
4. **Color Contrast**: Meet WCAG 2.1 AA standards
5. **Screen Reader Support**: Use semantic HTML and ARIA roles

### Examples

```tsx
// Icon button with aria-label
<Button variant="plain" aria-label="Edit model">
  <PencilIcon />
</Button>

// Table with aria-label
<Table aria-label="List of registered models">

// Live region for dynamic content
<div aria-live="polite">
  {notification && <Alert title={notification.title} />}
</div>
```

## Responsive Design

### Breakpoints

| Breakpoint | Width | Use Case |
|------------|-------|----------|
| sm | 576px | Mobile |
| md | 768px | Tablet |
| lg | 992px | Desktop |
| xl | 1200px | Large desktop |
| 2xl | 1450px | Extra large |

### Grid Layout

```tsx
<Grid hasGutter>
  <GridItem span={12} md={6} lg={4}>
    <Card>...</Card>
  </GridItem>
  <GridItem span={12} md={6} lg={4}>
    <Card>...</Card>
  </GridItem>
  <GridItem span={12} md={6} lg={4}>
    <Card>...</Card>
  </GridItem>
</Grid>
```

## Theme Configuration

### CSS Variables

```css
/* PatternFly CSS variables */
:root {
  --pf-v5-global--primary-color--100: #0066cc;
  --pf-v5-global--BackgroundColor--100: #ffffff;
  --pf-v5-global--BorderColor--100: #d2d2d2;
}
```

### Dark Mode

Support for dark mode through PatternFly's theme system:

```tsx
<Page className={isDarkMode ? 'pf-v5-theme-dark' : ''}>
```

## Icons

### PatternFly Icons

```tsx
import {
  CubesIcon,
  PlusCircleIcon,
  TrashIcon,
  EditIcon,
  SearchIcon,
} from '@patternfly/react-icons';

// Usage
<Button variant="primary">
  <PlusCircleIcon /> Register Model
</Button>
```

### Icon Guidelines

- Use consistent icon sizes
- Pair icons with text for clarity
- Use tooltips for icon-only buttons
- Follow PatternFly icon naming conventions

## Component Library

### Shared Components

Located in `clients/ui/frontend/src/app/components/`:

| Component | Purpose |
|-----------|---------|
| `DashboardSearchField` | Search with filters |
| `SimpleSelect` | Dropdown selection |
| `TableBase` | Base table component |
| `ApplicationsPage` | Standard page layout |
| `EmptyStateErrorMessage` | Error display |
| `NameDescriptionField` | Common form field |

### Domain Components

Model Registry specific components:

| Component | Purpose |
|-----------|---------|
| `RegisteredModelTable` | Model list display |
| `ModelVersionsTable` | Version list |
| `ModelArtifactList` | Artifact display |
| `RegisterModelModal` | Model registration form |

---

[Back to Guides Index](./README.md) | [Previous: Style Guide](./style-guide.md)

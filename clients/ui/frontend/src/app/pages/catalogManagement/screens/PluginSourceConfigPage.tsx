import * as React from 'react';
import {
  ActionList,
  ActionListGroup,
  ActionListItem,
  Alert,
  Breadcrumb,
  BreadcrumbItem,
  Button,
  Checkbox,
  EmptyState,
  EmptyStateBody,
  EmptyStateVariant,
  FileUpload,
  Form,
  FormGroup,
  FormHelperText,
  HelperText,
  HelperTextItem,
  PageSection,
  Radio,
  Sidebar,
  SidebarContent,
  SidebarPanel,
  Stack,
  StackItem,
  TextInput,
  Title,
} from '@patternfly/react-core';
import { CubesIcon } from '@patternfly/react-icons';
import { useNavigate, useParams } from 'react-router-dom';
import { CatalogManagementContext } from '~/app/context/catalogManagement/CatalogManagementContext';
import {
  catalogManagementUrl,
  catalogPluginSourcesUrl,
} from '~/app/routes/catalogManagement/catalogManagement';

const PLUGIN_LABELS: Record<string, { entityLabel: string; catalogLabel: string; description: string }> = {
  model: {
    entityLabel: 'models',
    catalogLabel: 'Model catalog',
    description: 'Add a new model source to your organization.',
  },
  mcp: {
    entityLabel: 'servers',
    catalogLabel: 'MCP catalog',
    description: 'Add a new MCP server source to your organization.',
  },
};

const PluginSourceConfigPage: React.FC = () => {
  const navigate = useNavigate();
  const { pluginName, sourceId } = useParams<{ pluginName: string; sourceId: string }>();
  const { apiState } = React.useContext(CatalogManagementContext);

  const isManageMode = !!sourceId;
  const effectivePluginName = pluginName || '';
  const labels = PLUGIN_LABELS[effectivePluginName] || {
    entityLabel: 'entities',
    catalogLabel: `${effectivePluginName} catalog`,
    description: `Add a new source to your organization.`,
  };

  const [sourceName, setSourceName] = React.useState('');
  const [sourceType, setSourceType] = React.useState('yaml');
  const [yamlContent, setYamlContent] = React.useState('');
  const [yamlFilename, setYamlFilename] = React.useState('');
  const [visibleInCatalog, setVisibleInCatalog] = React.useState(true);
  const [submitting, setSubmitting] = React.useState(false);
  const [submitError, setSubmitError] = React.useState<string | undefined>();
  const [sourceLoaded, setSourceLoaded] = React.useState(!isManageMode);

  React.useEffect(() => {
    if (!isManageMode || !apiState.apiAvailable || !pluginName) {
      return;
    }
    apiState.api
      .getPluginSources({}, pluginName)
      .then((data) => {
        const source = data.sources?.find((s) => s.id === sourceId);
        if (source) {
          setSourceName(source.name);
          setSourceType(source.type);
          setVisibleInCatalog(source.enabled);
          // Load YAML content from properties
          const content = source.properties?.content;
          if (typeof content === 'string') {
            setYamlContent(content);
          }
        }
        setSourceLoaded(true);
      })
      .catch(() => {
        setSourceLoaded(true);
      });
  }, [isManageMode, apiState, pluginName, sourceId]);

  const handleSubmit = React.useCallback(async () => {
    if (!apiState.apiAvailable || !pluginName || !sourceName.trim()) {
      return;
    }
    setSubmitting(true);
    setSubmitError(undefined);
    try {
      const payload = {
        id: isManageMode && sourceId ? sourceId : sourceName.toLowerCase().replace(/\s+/g, '-'),
        name: sourceName,
        type: sourceType,
        enabled: visibleInCatalog,
        properties: yamlContent ? { content: yamlContent } : undefined,
      };
      await apiState.api.applyPluginSource({}, pluginName, payload);
      navigate(catalogPluginSourcesUrl(pluginName));
    } catch (err) {
      setSubmitError(err instanceof Error ? err.message : String(err));
    } finally {
      setSubmitting(false);
    }
  }, [apiState, pluginName, sourceName, sourceType, yamlContent, visibleInCatalog, isManageMode, sourceId, navigate]);

  const handleFileChange = (
    _event: React.DragEvent<HTMLElement> | React.ChangeEvent<HTMLInputElement> | Event,
    file: File,
  ) => {
    setYamlFilename(file.name);
    const reader = new FileReader();
    reader.onload = () => {
      const text = typeof reader.result === 'string' ? reader.result : '';
      setYamlContent(text);
    };
    reader.readAsText(file);
  };

  const handleTextChange = (_event: React.ChangeEvent<HTMLTextAreaElement>, value: string) => {
    setYamlContent(value);
  };

  const handleClearFile = () => {
    setYamlFilename('');
    setYamlContent('');
  };

  const pageTitle = isManageMode ? 'Manage source' : 'Add a source';
  const pageDescription = isManageMode
    ? `Update the configuration for this ${labels.entityLabel} source.`
    : labels.description;

  const settingsLabel = `${labels.catalogLabel} settings`;

  if (!sourceLoaded) {
    return null;
  }

  return (
    <>
      <PageSection type="breadcrumb">
        <Breadcrumb>
          <BreadcrumbItem
            to={catalogPluginSourcesUrl(effectivePluginName)}
            onClick={(e) => {
              e.preventDefault();
              navigate(catalogPluginSourcesUrl(effectivePluginName));
            }}
          >
            {settingsLabel}
          </BreadcrumbItem>
          <BreadcrumbItem isActive>{pageTitle}</BreadcrumbItem>
        </Breadcrumb>
      </PageSection>
      <PageSection>
        <Title headingLevel="h1">{pageTitle}</Title>
        <p className="pf-v6-u-color-200 pf-v6-u-mt-sm">{pageDescription}</p>
      </PageSection>
      <PageSection isFilled>
        <Sidebar hasBorder isPanelRight hasGutter>
          <SidebarContent>
            <Form
              isWidthLimited
              onSubmit={(e) => {
                e.preventDefault();
                handleSubmit();
              }}
            >
              <Stack hasGutter>
                <StackItem>
                  <FormGroup label="Name" isRequired fieldId="source-name">
                    <TextInput
                      id="source-name"
                      isRequired
                      value={sourceName}
                      onChange={(_event, value) => setSourceName(value)}
                      data-testid="source-name-input"
                    />
                  </FormGroup>
                </StackItem>

                <StackItem>
                  <FormGroup label="Source type" fieldId="source-type" role="radiogroup">
                    <Radio
                      id="source-type-yaml"
                      name="source-type"
                      label="YAML file"
                      isChecked={sourceType === 'yaml'}
                      onChange={() => setSourceType('yaml')}
                      data-testid="source-type-yaml"
                    />
                    {effectivePluginName === 'model' && (
                      <Radio
                        id="source-type-huggingface"
                        name="source-type"
                        label="Hugging Face"
                        isChecked={sourceType === 'huggingface'}
                        onChange={() => setSourceType('huggingface')}
                        data-testid="source-type-huggingface"
                      />
                    )}
                  </FormGroup>
                </StackItem>

                {sourceType === 'yaml' && (
                  <StackItem>
                    <FormGroup
                      label="Upload a YAML file"
                      isRequired
                      fieldId="yaml-content"
                    >
                      <FileUpload
                        id="yaml-content"
                        data-testid="yaml-content-input"
                        type="text"
                        value={yamlContent}
                        filename={yamlFilename}
                        filenamePlaceholder="Drag and drop a YAML file or upload one"
                        onFileInputChange={handleFileChange}
                        onTextChange={handleTextChange}
                        onClearClick={handleClearFile}
                        browseButtonText="Upload"
                        allowEditingUploadedText
                        dropzoneProps={{
                          accept: { 'text/yaml': ['.yaml', '.yml'] },
                        }}
                      />
                      <FormHelperText>
                        <HelperText>
                          <HelperTextItem>Upload or paste a YAML string.</HelperTextItem>
                        </HelperText>
                      </FormHelperText>
                    </FormGroup>
                  </StackItem>
                )}

                <StackItem>
                  <FormGroup fieldId="visible-in-catalog">
                    <Checkbox
                      id="visible-in-catalog"
                      label={
                        <span className="pf-v6-c-form__label-text">Visible in catalog</span>
                      }
                      description={`When enabled, ${labels.entityLabel} from this source will appear in the ${labels.catalogLabel}.`}
                      isChecked={visibleInCatalog}
                      onChange={(_event, checked) => setVisibleInCatalog(checked)}
                      data-testid="visible-in-catalog-checkbox"
                    />
                  </FormGroup>
                </StackItem>
              </Stack>
            </Form>
          </SidebarContent>
          <SidebarPanel width={{ default: 'width_50' }}>
            <div data-testid="preview-panel" className="pf-v6-u-h-100">
              <Title headingLevel="h2" size="lg" className="pf-v6-u-mb-md">
                {labels.catalogLabel} preview
              </Title>
              <EmptyState
                icon={CubesIcon}
                titleText={`Preview ${labels.entityLabel}`}
                variant={EmptyStateVariant.sm}
              >
                <EmptyStateBody>
                  To view the {labels.entityLabel} from this source that will appear in the{' '}
                  {labels.catalogLabel.toLowerCase()} with your current configuration, complete all
                  required fields, then click <strong>Preview</strong>.
                </EmptyStateBody>
              </EmptyState>
            </div>
          </SidebarPanel>
        </Sidebar>
      </PageSection>
      <PageSection hasBodyWrapper={false} stickyOnBreakpoint={{ default: 'bottom' }}>
        <Stack hasGutter>
          {submitError && (
            <StackItem>
              <Alert variant="danger" isInline title="Error saving source">
                {submitError}
              </Alert>
            </StackItem>
          )}
          <StackItem>
            <ActionList>
              <ActionListGroup>
                <ActionListItem>
                  <Button
                    variant="primary"
                    onClick={handleSubmit}
                    isDisabled={submitting || !sourceName.trim()}
                    isLoading={submitting}
                    data-testid="submit-source-button"
                  >
                    {isManageMode ? 'Save' : 'Add'}
                  </Button>
                </ActionListItem>
                <ActionListItem>
                  <Button
                    variant="link"
                    onClick={() => navigate(catalogPluginSourcesUrl(effectivePluginName))}
                    data-testid="cancel-source-button"
                  >
                    Cancel
                  </Button>
                </ActionListItem>
              </ActionListGroup>
            </ActionList>
          </StackItem>
        </Stack>
      </PageSection>
    </>
  );
};

export default PluginSourceConfigPage;

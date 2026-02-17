import * as React from 'react';
import { useNavigate } from 'react-router-dom';
import {
  Card,
  CardBody,
  CardTitle,
  EmptyState,
  Gallery,
  PageSection,
  Spinner,
  Stack,
  StackItem,
  Title,
  Content,
} from '@patternfly/react-core';
import { ApplicationsPage } from 'mod-arch-shared';
import { useCatalogPlugins } from '~/app/context/catalog/CatalogContext';

const CatalogHomePage: React.FC = () => {
  const { plugins, pluginsLoaded, pluginsLoadError, capabilitiesMap } = useCatalogPlugins();
  const navigate = useNavigate();

  const handlePluginClick = (pluginName: string) => {
    // Navigate to the first entity type for this plugin
    const caps = capabilitiesMap[pluginName];
    if (caps && caps.entities.length > 0) {
      navigate(`/catalog/${pluginName}/${caps.entities[0].plural}`);
    } else {
      navigate(`/catalog/${pluginName}`);
    }
  };

  return (
    <ApplicationsPage
      title={<Title headingLevel="h1">Catalog</Title>}
      description="Discover and browse assets from all registered catalog plugins."
      loaded={pluginsLoaded}
      loadError={pluginsLoadError}
      errorMessage="Unable to load catalog plugins."
      empty={pluginsLoaded && plugins.length === 0}
      provideChildrenPadding
    >
      <Stack hasGutter>
        <StackItem>
          {!pluginsLoaded ? (
            <EmptyState>
              <Spinner />
              <Title headingLevel="h4" size="lg">
                Loading plugins...
              </Title>
            </EmptyState>
          ) : (
            <PageSection>
              <Gallery hasGutter minWidths={{ default: '300px' }}>
                {plugins.map((plugin) => {
                  const caps = capabilitiesMap[plugin.name];
                  const entityCount = caps ? caps.entities.length : 0;
                  return (
                    <Card
                      key={plugin.name}
                      isClickable
                      isSelectable
                      onClick={() => handlePluginClick(plugin.name)}
                    >
                      <CardTitle>{caps?.plugin.displayName || plugin.displayName || plugin.name}</CardTitle>
                      <CardBody>
                        <Content>
                          <p>{plugin.description}</p>
                          <p>
                            <strong>Version:</strong> {plugin.version}
                            {entityCount > 0 && (
                              <>
                                {' '}
                                | <strong>Entity types:</strong> {entityCount}
                              </>
                            )}
                          </p>
                        </Content>
                      </CardBody>
                    </Card>
                  );
                })}
              </Gallery>
            </PageSection>
          )}
        </StackItem>
      </Stack>
    </ApplicationsPage>
  );
};

export default CatalogHomePage;

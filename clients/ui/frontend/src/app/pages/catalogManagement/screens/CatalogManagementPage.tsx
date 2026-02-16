import * as React from 'react';
import {
  Card,
  CardBody,
  CardTitle,
  Gallery,
  GalleryItem,
  Label,
  LabelGroup,
  PageSection,
  Spinner,
  Title,
} from '@patternfly/react-core';
import { CheckCircleIcon, ExclamationCircleIcon } from '@patternfly/react-icons';
import { useNavigate } from 'react-router-dom';
import { ApplicationsPage } from 'mod-arch-shared';
import { CatalogManagementContext } from '~/app/context/catalogManagement/CatalogManagementContext';
import {
  CATALOG_MANAGEMENT_PAGE_TITLE,
  CATALOG_MANAGEMENT_DESCRIPTION,
  catalogPluginUrl,
} from '~/app/routes/catalogManagement/catalogManagement';
import { CatalogPluginInfo } from '~/app/catalogManagementTypes';

const CatalogManagementPage: React.FC = () => {
  const navigate = useNavigate();
  const { plugins, pluginsLoaded, pluginsLoadError } =
    React.useContext(CatalogManagementContext);

  const handlePluginClick = React.useCallback(
    (plugin: CatalogPluginInfo) => {
      navigate(catalogPluginUrl(plugin.name));
    },
    [navigate],
  );

  return (
    <ApplicationsPage
      title={<Title headingLevel="h1">{CATALOG_MANAGEMENT_PAGE_TITLE}</Title>}
      description={CATALOG_MANAGEMENT_DESCRIPTION}
      loaded={pluginsLoaded}
      loadError={pluginsLoadError}
      errorMessage="Unable to load catalog plugins."
      empty={pluginsLoaded && plugins.length === 0}
      provideChildrenPadding
    >
      <PageSection>
        <Gallery hasGutter minWidths={{ default: '300px' }}>
          {!pluginsLoaded ? (
            <Spinner size="lg" />
          ) : (
            plugins.map((plugin) => (
              <GalleryItem key={plugin.name}>
                <Card
                  isClickable
                  isSelectable
                  onClick={() => handlePluginClick(plugin)}
                  data-testid={`plugin-card-${plugin.name}`}
                >
                  <CardTitle>
                    {plugin.name}{' '}
                    {plugin.healthy ? (
                      <CheckCircleIcon color="var(--pf-t--global--color--status--success--default)" />
                    ) : (
                      <ExclamationCircleIcon color="var(--pf-t--global--color--status--danger--default)" />
                    )}
                  </CardTitle>
                  <CardBody>
                    <p>{plugin.description}</p>
                    <p>
                      <strong>Version:</strong> {plugin.version}
                    </p>
                    {plugin.entityKinds && plugin.entityKinds.length > 0 && (
                      <LabelGroup categoryName="Entity types">
                        {plugin.entityKinds.map((kind) => (
                          <Label key={kind} color="blue">
                            {kind}
                          </Label>
                        ))}
                      </LabelGroup>
                    )}
                    {plugin.management && (
                      <LabelGroup categoryName="Capabilities">
                        {plugin.management.sourceManager && (
                          <Label color="green">Sources</Label>
                        )}
                        {plugin.management.refresh && <Label color="green">Refresh</Label>}
                        {plugin.management.diagnostics && (
                          <Label color="green">Diagnostics</Label>
                        )}
                      </LabelGroup>
                    )}
                  </CardBody>
                </Card>
              </GalleryItem>
            ))
          )}
        </Gallery>
      </PageSection>
    </ApplicationsPage>
  );
};

export default CatalogManagementPage;

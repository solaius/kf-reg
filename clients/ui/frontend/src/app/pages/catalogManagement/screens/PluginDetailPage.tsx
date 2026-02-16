import * as React from 'react';
import {
  Breadcrumb,
  BreadcrumbItem,
  PageSection,
  Tab,
  Tabs,
  TabTitleText,
  Title,
} from '@patternfly/react-core';
import { Outlet, useNavigate, useParams, useLocation } from 'react-router-dom';
import { ApplicationsPage } from 'mod-arch-shared';
import { CatalogManagementContext } from '~/app/context/catalogManagement/CatalogManagementContext';
import { catalogManagementUrl } from '~/app/routes/catalogManagement/catalogManagement';
import { CatalogPluginInfo } from '~/app/catalogManagementTypes';

enum PluginTab {
  SOURCES = 'sources',
  DIAGNOSTICS = 'diagnostics',
}

const PluginDetailPage: React.FC = () => {
  const navigate = useNavigate();
  const location = useLocation();
  const { pluginName } = useParams<{ pluginName: string }>();
  const { plugins, pluginsLoaded, setSelectedPlugin } =
    React.useContext(CatalogManagementContext);

  const plugin = React.useMemo(
    () => plugins.find((p: CatalogPluginInfo) => p.name === pluginName),
    [plugins, pluginName],
  );

  React.useEffect(() => {
    setSelectedPlugin(plugin);
    return () => setSelectedPlugin(undefined);
  }, [plugin, setSelectedPlugin]);

  const activeTab = React.useMemo(() => {
    if (location.pathname.endsWith('/diagnostics')) {
      return PluginTab.DIAGNOSTICS;
    }
    return PluginTab.SOURCES;
  }, [location.pathname]);

  const handleTabSelect = React.useCallback(
    (_event: React.MouseEvent<HTMLElement>, tabKey: string | number) => {
      navigate(`${tabKey}`);
    },
    [navigate],
  );

  if (!pluginsLoaded) {
    return null;
  }

  if (!plugin) {
    return (
      <ApplicationsPage
        title={<Title headingLevel="h1">Plugin not found</Title>}
        description={`No plugin named "${pluginName}" was found.`}
        loaded
        empty={false}
        provideChildrenPadding
      />
    );
  }

  const hasDiagnostics = plugin.management?.diagnostics;

  return (
    <>
      <PageSection type="breadcrumb">
        <Breadcrumb>
          <BreadcrumbItem
            to={catalogManagementUrl()}
            onClick={(e) => {
              e.preventDefault();
              navigate(catalogManagementUrl());
            }}
          >
            Catalog Management
          </BreadcrumbItem>
          <BreadcrumbItem isActive>{plugin.name}</BreadcrumbItem>
        </Breadcrumb>
      </PageSection>
      <PageSection>
        <Title headingLevel="h1">
          {plugin.name} ({plugin.version})
        </Title>
        <p>{plugin.description}</p>
      </PageSection>
      <PageSection type="tabs">
        <Tabs activeKey={activeTab} onSelect={handleTabSelect}>
          <Tab
            eventKey={PluginTab.SOURCES}
            title={<TabTitleText>Sources</TabTitleText>}
            data-testid="plugin-sources-tab"
          />
          {hasDiagnostics && (
            <Tab
              eventKey={PluginTab.DIAGNOSTICS}
              title={<TabTitleText>Diagnostics</TabTitleText>}
              data-testid="plugin-diagnostics-tab"
            />
          )}
        </Tabs>
      </PageSection>
      <PageSection isFilled>
        <Outlet />
      </PageSection>
    </>
  );
};

export default PluginDetailPage;

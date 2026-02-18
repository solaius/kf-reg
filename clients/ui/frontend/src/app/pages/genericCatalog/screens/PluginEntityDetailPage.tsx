import * as React from 'react';
import { useParams } from 'react-router-dom';
import {
  Breadcrumb,
  BreadcrumbItem,
  EmptyState,
  PageSection,
  Spinner,
  Stack,
  StackItem,
  Title,
} from '@patternfly/react-core';
import { ApplicationsPage } from 'mod-arch-shared';
import { useNotification } from '~/app/hooks/useNotification';
import { useCatalogPlugins } from '~/app/context/catalog/CatalogContext';
import { getEntity, executeEntityAction } from '~/app/api/catalogEntities/service';
import { GenericEntity } from '~/app/types/asset';
import { ActionDefinition } from '~/app/types/capabilities';
import { BFF_API_VERSION, URL_PREFIX } from '~/app/utilities/const';
import GenericDetailView from '~/app/pages/genericCatalog/components/GenericDetailView';
import GenericActionBar from '~/app/pages/genericCatalog/components/GenericActionBar';
import GenericActionDialog from '~/app/pages/genericCatalog/components/GenericActionDialog';
import GovernancePanel from '~/app/pages/genericCatalog/components/GovernancePanel';
import VersionsPanel from '~/app/pages/genericCatalog/components/VersionsPanel';
import PromotionPanel from '~/app/pages/genericCatalog/components/PromotionPanel';
import ApprovalsPanel from '~/app/pages/genericCatalog/components/ApprovalsPanel';
import AuditHistoryPanel from '~/app/pages/genericCatalog/components/AuditHistoryPanel';
import ProvenancePanel from '~/app/pages/genericCatalog/components/ProvenancePanel';
import { getFieldValue } from '~/app/pages/genericCatalog/utils';
import { GovernanceResponse } from '~/app/types/governance';
import { getGovernance } from '~/app/api/governance/service';

const PluginEntityDetailPage: React.FC = () => {
  const { pluginName = '', entityPlural = '', entityName = '' } = useParams<{
    pluginName: string;
    entityPlural: string;
    entityName: string;
  }>();
  const { getPluginCaps } = useCatalogPlugins();
  const notification = useNotification();

  const hostPath = `${URL_PREFIX}/api/${BFF_API_VERSION}/model_catalog`;

  const caps = getPluginCaps(pluginName);
  const entityCaps = caps?.entities.find((e) => e.plural === entityPlural);

  const [data, setData] = React.useState<GenericEntity | undefined>();
  const [loaded, setLoaded] = React.useState(false);
  const [error, setError] = React.useState<Error | undefined>();

  const [activeAction, setActiveAction] = React.useState<ActionDefinition | undefined>();
  const [governanceData, setGovernanceData] = React.useState<GovernanceResponse | undefined>();

  const governanceSupported = entityCaps?.governance?.supported === true;

  // Fetch entity data
  React.useEffect(() => {
    if (!pluginName || !entityPlural || !entityName) {
      return;
    }

    setLoaded(false);
    const fetcher = getEntity(hostPath, pluginName, entityPlural, entityName);
    fetcher({})
      .then((result: GenericEntity) => {
        setData(result);
        setLoaded(true);
        setError(undefined);
      })
      .catch((err: Error) => {
        setError(err);
        setLoaded(true);
      });
  }, [hostPath, pluginName, entityPlural, entityName]);

  // Fetch governance data when governance is supported
  React.useEffect(() => {
    if (!governanceSupported || !pluginName || !entityPlural || !entityName) {
      return;
    }
    // Derive kind from plural (strip trailing 's' as a simple heuristic)
    const kind = entityCaps?.kind || entityPlural;
    const govFetcher = getGovernance(pluginName, kind, entityName);
    govFetcher({})
      .then((result: GovernanceResponse) => {
        setGovernanceData(result);
      })
      .catch(() => {
        // Governance fetch failure is non-fatal
      });
  }, [governanceSupported, pluginName, entityPlural, entityName, entityCaps?.kind]);

  // Resolve available actions for this entity
  const availableActions = React.useMemo(() => {
    if (!entityCaps?.actions || !caps?.actions) {
      return [];
    }
    return caps.actions.filter((a) => entityCaps.actions?.includes(a.id));
  }, [entityCaps, caps]);

  const handleActionClick = (actionId: string) => {
    const action = availableActions.find((a) => a.id === actionId);
    if (action) {
      setActiveAction(action);
    }
  };

  const handleActionExecute = async (params: Record<string, unknown>): Promise<void> => {
    if (!activeAction) {
      return;
    }
    const executor = executeEntityAction(
      hostPath,
      pluginName,
      entityPlural,
      entityName,
      activeAction.id,
      params,
    );
    const result: GenericEntity = await executor({});
    setData(result);
    notification.success(`${activeAction.displayName} completed`);
  };

  const displayName = entityCaps?.displayName || entityPlural;
  const nameField = entityCaps?.uiHints?.nameField || 'name';
  const entityDisplayName = data ? String(getFieldValue(data, nameField) || entityName) : entityName;

  return (
    <ApplicationsPage
      title={<Title headingLevel="h1">{entityDisplayName}</Title>}
      breadcrumb={
        <Breadcrumb>
          <BreadcrumbItem to="/catalog">Catalog</BreadcrumbItem>
          <BreadcrumbItem to={`/catalog/${pluginName}/${entityPlural}`}>
            {displayName}
          </BreadcrumbItem>
          <BreadcrumbItem isActive>{entityDisplayName}</BreadcrumbItem>
        </Breadcrumb>
      }
      loaded={loaded}
      loadError={error}
      errorMessage={`Unable to load ${entityDisplayName}.`}
      empty={loaded && !data}
      provideChildrenPadding
    >
      {data && entityCaps ? (
        <Stack hasGutter>
          {availableActions.length > 0 && (
            <StackItem>
              <GenericActionBar
                actions={availableActions}
                onActionClick={handleActionClick}
              />
            </StackItem>
          )}
          <StackItem isFilled>
            <PageSection padding={{ default: 'noPadding' }}>
              <GenericDetailView entity={entityCaps} data={data} />
            </PageSection>
          </StackItem>
          {governanceSupported && governanceData && (
            <StackItem>
              <GovernancePanel governance={governanceData.governance} />
            </StackItem>
          )}
          {governanceSupported && entityCaps?.governance?.versioning?.enabled && (
            <StackItem>
              <VersionsPanel
                plugin={pluginName}
                kind={entityCaps.kind || entityPlural}
                name={entityName}
              />
            </StackItem>
          )}
          {governanceSupported && entityCaps?.governance?.versioning?.enabled && (
            <StackItem>
              <ProvenancePanel
                plugin={pluginName}
                kind={entityCaps.kind || entityPlural}
                name={entityName}
              />
            </StackItem>
          )}
          {governanceSupported && entityCaps?.governance?.versioning?.enabled && (
            <StackItem>
              <PromotionPanel
                plugin={pluginName}
                kind={entityCaps.kind || entityPlural}
                name={entityName}
              />
            </StackItem>
          )}
          {governanceSupported && entityCaps?.governance?.approvals?.enabled && (
            <StackItem>
              <ApprovalsPanel
                plugin={pluginName}
                kind={entityCaps.kind || entityPlural}
                name={entityName}
              />
            </StackItem>
          )}
          {governanceSupported && (
            <StackItem>
              <AuditHistoryPanel
                plugin={pluginName}
                kind={entityCaps.kind || entityPlural}
                name={entityName}
              />
            </StackItem>
          )}
          {activeAction && (
            <GenericActionDialog
              action={activeAction}
              isOpen={Boolean(activeAction)}
              onClose={() => setActiveAction(undefined)}
              onExecute={handleActionExecute}
            />
          )}
        </Stack>
      ) : (
        <EmptyState titleText="Entity not found" headingLevel="h4">
          <p>The requested entity could not be found.</p>
        </EmptyState>
      )}
    </ApplicationsPage>
  );
};

export default PluginEntityDetailPage;

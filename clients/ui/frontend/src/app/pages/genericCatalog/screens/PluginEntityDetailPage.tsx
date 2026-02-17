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
import { getFieldValue } from '~/app/pages/genericCatalog/utils';

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

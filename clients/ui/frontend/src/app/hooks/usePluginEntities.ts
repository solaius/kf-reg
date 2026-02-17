import * as React from 'react';
import { getEntityList } from '~/app/api/catalogEntities/service';
import { GenericEntity, GenericEntityList } from '~/app/types/asset';
import { BFF_API_VERSION, URL_PREFIX } from '~/app/utilities/const';

type UsePluginEntitiesResult = {
  entities: GenericEntity[];
  loaded: boolean;
  error?: Error;
  totalSize: number;
  nextPageToken?: string;
  refresh: () => void;
  loadMore: () => void;
  isLoadingMore: boolean;
};

export const usePluginEntities = (
  pluginName: string,
  entityPlural: string,
  queryParams?: Record<string, unknown>,
): UsePluginEntitiesResult => {
  const hostPath = `${URL_PREFIX}/api/${BFF_API_VERSION}/model_catalog`;

  const [entities, setEntities] = React.useState<GenericEntity[]>([]);
  const [loaded, setLoaded] = React.useState(false);
  const [error, setError] = React.useState<Error | undefined>();
  const [totalSize, setTotalSize] = React.useState(0);
  const [nextPageToken, setNextPageToken] = React.useState<string | undefined>();
  const [refreshCounter, setRefreshCounter] = React.useState(0);
  const [isLoadingMore, setIsLoadingMore] = React.useState(false);

  const refresh = React.useCallback(() => {
    setRefreshCounter((c) => c + 1);
  }, []);

  React.useEffect(() => {
    if (!pluginName || !entityPlural) {
      return;
    }

    setLoaded(false);
    const fetcher = getEntityList(hostPath, pluginName, entityPlural, queryParams);
    fetcher({})
      .then((data: GenericEntityList) => {
        setEntities(data.items || []);
        setTotalSize(data.size || 0);
        setNextPageToken(data.nextPageToken);
        setLoaded(true);
        setError(undefined);
      })
      .catch((err: Error) => {
        setError(err);
        setLoaded(true);
      });
  }, [hostPath, pluginName, entityPlural, queryParams, refreshCounter]);

  const loadMore = React.useCallback(() => {
    if (!nextPageToken) {
      return;
    }
    setIsLoadingMore(true);
    const moreParams = { ...queryParams, nextPageToken };
    const fetcher = getEntityList(hostPath, pluginName, entityPlural, moreParams);
    fetcher({})
      .then((data: GenericEntityList) => {
        setEntities((prev) => [...prev, ...(data.items || [])]);
        setTotalSize(data.size || 0);
        setNextPageToken(data.nextPageToken);
      })
      .catch((err: Error) => {
        setError(err);
      })
      .finally(() => {
        setIsLoadingMore(false);
      });
  }, [hostPath, pluginName, entityPlural, queryParams, nextPageToken]);

  return { entities, loaded, error, totalSize, nextPageToken, refresh, loadMore, isLoadingMore };
};

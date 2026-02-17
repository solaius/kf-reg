import * as React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { CatalogContextProvider } from '~/app/context/catalog/CatalogContext';
import CatalogHomePage from './screens/CatalogHomePage';
import PluginEntityListPage from './screens/PluginEntityListPage';
import PluginEntityDetailPage from './screens/PluginEntityDetailPage';

const GenericCatalogRoutes: React.FC = () => (
  <CatalogContextProvider>
    <Routes>
      <Route index element={<CatalogHomePage />} />
      <Route path=":pluginName/:entityPlural" element={<PluginEntityListPage />} />
      <Route path=":pluginName/:entityPlural/:entityName" element={<PluginEntityDetailPage />} />
      <Route path="*" element={<Navigate to="." replace />} />
    </Routes>
  </CatalogContextProvider>
);

export default GenericCatalogRoutes;

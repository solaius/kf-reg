import * as React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { CatalogManagementContextProvider } from '~/app/context/catalogManagement/CatalogManagementContext';
import CatalogManagementPage from './screens/CatalogManagementPage';
import PluginDetailPage from './screens/PluginDetailPage';
import PluginSourcesPage from './screens/PluginSourcesPage';
import PluginSourceConfigPage from './screens/PluginSourceConfigPage';
import PluginDiagnosticsPage from './screens/PluginDiagnosticsPage';

const CatalogManagementRoutes: React.FC = () => (
  <CatalogManagementContextProvider>
    <Routes>
      <Route index element={<CatalogManagementPage />} />
      <Route path="plugin/:pluginName" element={<PluginDetailPage />}>
        <Route index element={<Navigate to="sources" replace />} />
        <Route path="sources" element={<PluginSourcesPage />} />
        <Route path="diagnostics" element={<PluginDiagnosticsPage />} />
      </Route>
      <Route path="plugin/:pluginName/sources/add" element={<PluginSourceConfigPage />} />
      <Route path="plugin/:pluginName/sources/:sourceId/manage" element={<PluginSourceConfigPage />} />
      <Route path="*" element={<Navigate to="." replace />} />
    </Routes>
  </CatalogManagementContextProvider>
);

export default CatalogManagementRoutes;

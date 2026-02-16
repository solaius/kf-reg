import * as React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { McpCatalogContextProvider } from '~/app/context/mcpCatalog/McpCatalogContext';
import McpCatalogPage from './screens/McpCatalogPage';
import McpServerDetailPage from './screens/McpServerDetailPage';

const McpCatalogRoutes: React.FC = () => (
  <McpCatalogContextProvider>
    <Routes>
      <Route index element={<McpCatalogPage />} />
      <Route path=":serverName/*" element={<McpServerDetailPage />} />
      <Route path="*" element={<Navigate to="." replace />} />
    </Routes>
  </McpCatalogContextProvider>
);

export default McpCatalogRoutes;

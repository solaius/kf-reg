import * as React from 'react';
import {
  Card,
  CardBody,
  CardTitle,
  ClipboardCopy,
  DescriptionList,
  DescriptionListDescription,
  DescriptionListGroup,
  DescriptionListTerm,
  Label,
  LabelGroup,
} from '@patternfly/react-core';
import { ExternalLinkAltIcon } from '@patternfly/react-icons';
import { McpServer } from '~/app/mcpCatalogTypes';

type McpServerDetailsSidebarProps = {
  server: McpServer;
};

const McpServerDetailsSidebar: React.FC<McpServerDetailsSidebarProps> = ({ server }) => (
  <Card>
    <CardTitle>Server Details</CardTitle>
    <CardBody>
      <DescriptionList>
        {server.tags && server.tags.length > 0 && (
          <DescriptionListGroup>
            <DescriptionListTerm>Tags</DescriptionListTerm>
            <DescriptionListDescription>
              <LabelGroup>
                {server.tags.map((tag) => (
                  <Label key={tag} isCompact>
                    {tag}
                  </Label>
                ))}
              </LabelGroup>
            </DescriptionListDescription>
          </DescriptionListGroup>
        )}
        {server.license && (
          <DescriptionListGroup>
            <DescriptionListTerm>License</DescriptionListTerm>
            <DescriptionListDescription>{server.license}</DescriptionListDescription>
          </DescriptionListGroup>
        )}
        {server.version && (
          <DescriptionListGroup>
            <DescriptionListTerm>Version</DescriptionListTerm>
            <DescriptionListDescription>{server.version}</DescriptionListDescription>
          </DescriptionListGroup>
        )}
        <DescriptionListGroup>
          <DescriptionListTerm>Deployment Mode</DescriptionListTerm>
          <DescriptionListDescription>
            <Label color={server.deploymentMode === 'local' ? 'blue' : 'orange'}>
              {server.deploymentMode || 'unknown'}
            </Label>
          </DescriptionListDescription>
        </DescriptionListGroup>
        {server.image && (
          <DescriptionListGroup>
            <DescriptionListTerm>Image</DescriptionListTerm>
            <DescriptionListDescription>
              <ClipboardCopy isReadOnly hoverTip="Copy" clickTip="Copied" variant="inline-compact">
                {server.image}
              </ClipboardCopy>
            </DescriptionListDescription>
          </DescriptionListGroup>
        )}
        {server.endpoint && (
          <DescriptionListGroup>
            <DescriptionListTerm>Endpoint</DescriptionListTerm>
            <DescriptionListDescription>
              <ClipboardCopy isReadOnly hoverTip="Copy" clickTip="Copied" variant="inline-compact">
                {server.endpoint}
              </ClipboardCopy>
            </DescriptionListDescription>
          </DescriptionListGroup>
        )}
        {server.sourceUrl && (
          <DescriptionListGroup>
            <DescriptionListTerm>Source Code</DescriptionListTerm>
            <DescriptionListDescription>
              <a href={server.sourceUrl} target="_blank" rel="noopener noreferrer">
                Repository <ExternalLinkAltIcon />
              </a>
            </DescriptionListDescription>
          </DescriptionListGroup>
        )}
        {server.provider && (
          <DescriptionListGroup>
            <DescriptionListTerm>Provider</DescriptionListTerm>
            <DescriptionListDescription>{server.provider}</DescriptionListDescription>
          </DescriptionListGroup>
        )}
        <DescriptionListGroup>
          <DescriptionListTerm>Transport</DescriptionListTerm>
          <DescriptionListDescription>
            {server.supportedTransports || server.transportType || 'N/A'}
          </DescriptionListDescription>
        </DescriptionListGroup>
        {server.lastModified && (
          <DescriptionListGroup>
            <DescriptionListTerm>Last Modified</DescriptionListTerm>
            <DescriptionListDescription>{server.lastModified}</DescriptionListDescription>
          </DescriptionListGroup>
        )}
      </DescriptionList>
    </CardBody>
  </Card>
);

export default McpServerDetailsSidebar;

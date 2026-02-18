import * as React from 'react';
import {
  Card,
  CardBody,
  CardTitle,
  DescriptionList,
  DescriptionListDescription,
  DescriptionListGroup,
  DescriptionListTerm,
  EmptyState,
  Label,
} from '@patternfly/react-core';
import { ProvenanceInfo, VersionListResponse } from '~/app/types/governance';
import { listVersions } from '~/app/api/governance/service';

type ProvenancePanelProps = {
  plugin: string;
  kind: string;
  name: string;
};

const ProvenanceDetail: React.FC<{ provenance: ProvenanceInfo }> = ({ provenance }) => (
  <DescriptionList>
    {provenance.sourceType && (
      <DescriptionListGroup>
        <DescriptionListTerm>Source Type</DescriptionListTerm>
        <DescriptionListDescription>{provenance.sourceType}</DescriptionListDescription>
      </DescriptionListGroup>
    )}
    {provenance.sourceUri && (
      <DescriptionListGroup>
        <DescriptionListTerm>Source URI</DescriptionListTerm>
        <DescriptionListDescription>
          <a href={provenance.sourceUri} target="_blank" rel="noopener noreferrer">
            {provenance.sourceUri}
          </a>
        </DescriptionListDescription>
      </DescriptionListGroup>
    )}
    {provenance.sourceId && (
      <DescriptionListGroup>
        <DescriptionListTerm>Source ID</DescriptionListTerm>
        <DescriptionListDescription>{provenance.sourceId}</DescriptionListDescription>
      </DescriptionListGroup>
    )}
    {provenance.revisionId && (
      <DescriptionListGroup>
        <DescriptionListTerm>Revision ID</DescriptionListTerm>
        <DescriptionListDescription>{provenance.revisionId}</DescriptionListDescription>
      </DescriptionListGroup>
    )}
    {provenance.observedAt && (
      <DescriptionListGroup>
        <DescriptionListTerm>Observed At</DescriptionListTerm>
        <DescriptionListDescription>
          {new Date(provenance.observedAt).toLocaleString()}
        </DescriptionListDescription>
      </DescriptionListGroup>
    )}
    {provenance.integrity && (
      <DescriptionListGroup>
        <DescriptionListTerm>Integrity</DescriptionListTerm>
        <DescriptionListDescription>
          <Label color={provenance.integrity.verified ? 'green' : 'red'}>
            {provenance.integrity.verified ? 'Verified' : 'Unverified'}
          </Label>
          {provenance.integrity.method && ` (${provenance.integrity.method})`}
        </DescriptionListDescription>
      </DescriptionListGroup>
    )}
  </DescriptionList>
);

const ProvenancePanel: React.FC<ProvenancePanelProps> = ({ plugin, kind, name }) => {
  const [provenance, setProvenance] = React.useState<ProvenanceInfo | undefined>();
  const [loaded, setLoaded] = React.useState(false);

  React.useEffect(() => {
    const fetcher = listVersions(plugin, kind, name);
    fetcher({})
      .then((result: VersionListResponse) => {
        // Get provenance from the latest version (first in list, ordered by newest).
        const latest = (result.versions || [])[0];
        if (latest?.provenance) {
          setProvenance(latest.provenance);
        }
        setLoaded(true);
      })
      .catch(() => setLoaded(true));
  }, [plugin, kind, name]);

  if (!loaded) {
    return null;
  }

  return (
    <Card>
      <CardTitle>Provenance</CardTitle>
      <CardBody>
        {provenance ? (
          <ProvenanceDetail provenance={provenance} />
        ) : (
          <EmptyState titleText="No provenance data" headingLevel="h4">
            <p>No provenance information is available for this asset.</p>
          </EmptyState>
        )}
      </CardBody>
    </Card>
  );
};

export default ProvenancePanel;

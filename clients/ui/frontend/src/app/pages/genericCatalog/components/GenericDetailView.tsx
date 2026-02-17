import * as React from 'react';
import {
  Card,
  CardBody,
  CardTitle,
  DescriptionList,
  DescriptionListDescription,
  DescriptionListGroup,
  DescriptionListTerm,
  Label,
  LabelGroup,
  Stack,
  StackItem,
} from '@patternfly/react-core';
import { EntityCapabilities, V2FieldHint } from '~/app/types/capabilities';
import { GenericEntity } from '~/app/types/asset';
import { getFieldValue, formatFieldValue } from '~/app/pages/genericCatalog/utils';

type GenericDetailViewProps = {
  entity: EntityCapabilities;
  data: GenericEntity;
};

const renderFieldValue = (value: unknown, hint: V2FieldHint): React.ReactNode => {
  if (value == null) {
    return '-';
  }
  if (hint.type === 'tags' && Array.isArray(value)) {
    return (
      <LabelGroup>
        {value.map((tag) => (
          <Label key={String(tag)}>{String(tag)}</Label>
        ))}
      </LabelGroup>
    );
  }
  if (hint.type === 'url' && typeof value === 'string') {
    return (
      <a href={value} target="_blank" rel="noopener noreferrer">
        {value}
      </a>
    );
  }
  if (hint.type === 'markdown' && typeof value === 'string') {
    return <pre style={{ whiteSpace: 'pre-wrap' }}>{value}</pre>;
  }
  return formatFieldValue(value, hint.type);
};

const GenericDetailView: React.FC<GenericDetailViewProps> = ({ entity, data }) => {
  const detailFields = entity.fields.detailFields || [];
  const sections = entity.uiHints?.detailSections || ['General'];

  // Group fields by section
  const fieldsBySection: Record<string, V2FieldHint[]> = {};
  for (const section of sections) {
    fieldsBySection[section] = [];
  }
  for (const field of detailFields) {
    const section = field.section || sections[0] || 'General';
    if (!fieldsBySection[section]) {
      fieldsBySection[section] = [];
    }
    fieldsBySection[section].push(field);
  }

  return (
    <Stack hasGutter>
      {Object.entries(fieldsBySection).map(([sectionName, fields]) => {
        if (fields.length === 0) {
          return null;
        }
        return (
          <StackItem key={sectionName}>
            <Card>
              <CardTitle>{sectionName}</CardTitle>
              <CardBody>
                <DescriptionList>
                  {fields.map((field) => {
                    const value = getFieldValue(data, field.path);
                    return (
                      <DescriptionListGroup key={field.path}>
                        <DescriptionListTerm>{field.displayName}</DescriptionListTerm>
                        <DescriptionListDescription>
                          {renderFieldValue(value, field)}
                        </DescriptionListDescription>
                      </DescriptionListGroup>
                    );
                  })}
                </DescriptionList>
              </CardBody>
            </Card>
          </StackItem>
        );
      })}
    </Stack>
  );
};

export default GenericDetailView;

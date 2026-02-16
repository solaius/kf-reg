import * as React from 'react';
import {
  Alert,
  DescriptionList,
  DescriptionListDescription,
  DescriptionListGroup,
  DescriptionListTerm,
  ExpandableSection,
  Label,
  Stack,
  StackItem,
} from '@patternfly/react-core';
import { DetailedValidationResult } from '~/app/catalogManagementTypes';

type ValidationResultPanelProps = {
  result: DetailedValidationResult;
};

const ValidationResultPanel: React.FC<ValidationResultPanelProps> = ({ result }) => {
  const [isExpanded, setIsExpanded] = React.useState(!result.valid);

  return (
    <ExpandableSection
      toggleText={result.valid ? 'Validation passed' : 'Validation failed'}
      onToggle={(_event, expanded) => setIsExpanded(expanded)}
      isExpanded={isExpanded}
      data-testid="validation-result-panel"
    >
      <Stack hasGutter>
        {result.errors.length > 0 && (
          <StackItem>
            {result.errors.map((err, idx) => (
              <Alert
                key={`error-${idx}`}
                variant="danger"
                isInline
                isPlain
                title={err.field ? `${err.field}: ${err.message}` : err.message}
                data-testid={`validation-error-${idx}`}
              />
            ))}
          </StackItem>
        )}

        {result.warnings && result.warnings.length > 0 && (
          <StackItem>
            {result.warnings.map((warn, idx) => (
              <Alert
                key={`warning-${idx}`}
                variant="warning"
                isInline
                isPlain
                title={warn.field ? `${warn.field}: ${warn.message}` : warn.message}
                data-testid={`validation-warning-${idx}`}
              />
            ))}
          </StackItem>
        )}

        {result.layerResults.length > 0 && (
          <StackItem>
            <DescriptionList isHorizontal isCompact data-testid="layer-results">
              {result.layerResults.map((layer) => (
                <DescriptionListGroup key={layer.layer}>
                  <DescriptionListTerm>{layer.layer}</DescriptionListTerm>
                  <DescriptionListDescription>
                    <Stack>
                      <StackItem>
                        <Label color={layer.valid ? 'green' : 'red'}>
                          {layer.valid ? 'Passed' : 'Failed'}
                        </Label>
                      </StackItem>
                      {layer.errors &&
                        layer.errors.map((err, idx) => (
                          <StackItem key={idx}>
                            <Alert
                              variant="danger"
                              isInline
                              isPlain
                              title={err.field ? `${err.field}: ${err.message}` : err.message}
                            />
                          </StackItem>
                        ))}
                    </Stack>
                  </DescriptionListDescription>
                </DescriptionListGroup>
              ))}
            </DescriptionList>
          </StackItem>
        )}
      </Stack>
    </ExpandableSection>
  );
};

export default ValidationResultPanel;

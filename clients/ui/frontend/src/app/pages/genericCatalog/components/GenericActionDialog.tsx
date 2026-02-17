import * as React from 'react';
import {
  Alert,
  Modal,
  ModalVariant,
  ModalHeader,
  ModalBody,
  ModalFooter,
  Button,
  Form,
  FormGroup,
  HelperText,
  HelperTextItem,
  TextInput,
  Checkbox,
} from '@patternfly/react-core';
import { ActionDefinition } from '~/app/types/capabilities';

type GenericActionDialogProps = {
  action: ActionDefinition;
  isOpen: boolean;
  onClose: () => void;
  onExecute: (params: Record<string, unknown>) => Promise<void>;
};

const GenericActionDialog: React.FC<GenericActionDialogProps> = ({
  action,
  isOpen,
  onClose,
  onExecute,
}) => {
  const [formValues, setFormValues] = React.useState<Record<string, unknown>>({});
  const [isSubmitting, setIsSubmitting] = React.useState(false);
  const [error, setError] = React.useState<string | undefined>();

  React.useEffect(() => {
    if (isOpen) {
      // Initialize with default values
      const defaults: Record<string, unknown> = {};
      (action.parameters || []).forEach((param) => {
        if (param.defaultValue !== undefined) {
          defaults[param.name] = param.defaultValue;
        } else if (param.type === 'boolean') {
          defaults[param.name] = false;
        } else {
          defaults[param.name] = '';
        }
      });
      setFormValues(defaults);
      setError(undefined);
    }
  }, [isOpen, action.parameters]);

  const handleSubmit = async () => {
    setIsSubmitting(true);
    try {
      await onExecute(formValues);
      onClose();
    } catch (e) {
      setError(e instanceof Error ? e.message : String(e));
    } finally {
      setIsSubmitting(false);
    }
  };

  const hasParameters = action.parameters && action.parameters.length > 0;

  return (
    <Modal
      variant={ModalVariant.small}
      isOpen={isOpen}
      onClose={onClose}
    >
      <ModalHeader title={action.displayName} description={action.description} />
      <ModalBody>
        {error && (
          <Alert variant="danger" isInline title="Action failed" className="pf-v6-u-mb-md">
            {error}
          </Alert>
        )}
        {hasParameters ? (
          <Form>
            {(action.parameters || []).map((param) => {
              if (param.type === 'boolean') {
                return (
                  <FormGroup key={param.name} label={param.label}>
                    <Checkbox
                      id={`action-param-${param.name}`}
                      label={param.description || param.label}
                      isChecked={Boolean(formValues[param.name])}
                      onChange={(_e, checked) =>
                        setFormValues((prev) => ({ ...prev, [param.name]: checked }))
                      }
                    />
                  </FormGroup>
                );
              }
              if (param.type === 'tags') {
                return (
                  <FormGroup
                    key={param.name}
                    label={param.label}
                    isRequired={param.required}
                  >
                    <TextInput
                      id={`action-param-${param.name}`}
                      value={String(formValues[param.name] || '')}
                      onChange={(_e, value) =>
                        setFormValues((prev) => ({ ...prev, [param.name]: value }))
                      }
                    />
                    <HelperText>
                      <HelperTextItem>Comma-separated values</HelperTextItem>
                    </HelperText>
                  </FormGroup>
                );
              }
              return (
                <FormGroup
                  key={param.name}
                  label={param.label}
                  isRequired={param.required}
                >
                  <TextInput
                    id={`action-param-${param.name}`}
                    type={param.type === 'number' ? 'number' : 'text'}
                    value={String(formValues[param.name] || '')}
                    onChange={(_e, value) =>
                      setFormValues((prev) => ({
                        ...prev,
                        [param.name]: param.type === 'number' ? Number(value) : value,
                      }))
                    }
                  />
                </FormGroup>
              );
            })}
          </Form>
        ) : (
          <p>
            {action.destructive
              ? `Are you sure you want to ${action.displayName.toLowerCase()}?`
              : `Execute ${action.displayName}?`}
          </p>
        )}
      </ModalBody>
      <ModalFooter>
        <Button
          variant={action.destructive ? 'danger' : 'primary'}
          onClick={handleSubmit}
          isLoading={isSubmitting}
          isDisabled={isSubmitting}
        >
          {action.destructive ? action.displayName : 'Execute'}
        </Button>
        <Button variant="link" onClick={onClose} isDisabled={isSubmitting}>
          Cancel
        </Button>
      </ModalFooter>
    </Modal>
  );
};

export default GenericActionDialog;

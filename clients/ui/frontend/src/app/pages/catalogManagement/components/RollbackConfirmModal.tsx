import * as React from 'react';
import {
  Button,
  Modal,
  ModalBody,
  ModalHeader,
  ModalFooter,
} from '@patternfly/react-core';

type RollbackConfirmModalProps = {
  version: string;
  onConfirm: () => void;
  onCancel: () => void;
  isSubmitting: boolean;
};

const RollbackConfirmModal: React.FC<RollbackConfirmModalProps> = ({
  version,
  onConfirm,
  onCancel,
  isSubmitting,
}) => (
  <Modal isOpen onClose={onCancel} variant="small" data-testid="rollback-confirm-modal">
    <ModalHeader title="Rollback Configuration" />
    <ModalBody>
      <p>
        Rolling back will replace the current configuration with the version{' '}
        <code>{version}</code>. Any unsaved changes will be lost.
      </p>
      <p className="pf-v6-u-mt-md">
        After rollback, the source will be refreshed automatically to apply the restored
        configuration.
      </p>
    </ModalBody>
    <ModalFooter>
      <Button
        variant="primary"
        onClick={onConfirm}
        isLoading={isSubmitting}
        isDisabled={isSubmitting}
        data-testid="rollback-confirm-button"
      >
        Rollback
      </Button>
      <Button variant="link" onClick={onCancel} data-testid="rollback-cancel-button">
        Cancel
      </Button>
    </ModalFooter>
  </Modal>
);

export default RollbackConfirmModal;

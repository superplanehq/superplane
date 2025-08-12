import { createPortal } from 'react-dom';
import {
  Dialog,
  DialogTitle,
  DialogDescription,
  DialogBody,
  DialogActions,
} from '../Dialog/dialog';
import { Button } from '../Button/button';

interface IntegrationModalProps {
  open: boolean;
  onClose: () => void;
  integrationType: string;
}

export function IntegrationModal({ open, onClose, integrationType }: IntegrationModalProps) {
  const getIntegrationDisplayName = () => {
    switch (integrationType) {
      case 'semaphore':
        return 'Semaphore';
      case 'github':
        return 'GitHub';
      default:
        return integrationType;
    }
  };

  if (!open) return null;

  return createPortal(
    <Dialog open={open} onClose={() => { }} className="relative z-[9999]" size="md">
      <DialogTitle>Create {getIntegrationDisplayName()} Integration</DialogTitle>
      <DialogDescription>
        Integration creation is coming soon. You'll be able to create and manage integrations directly from this modal.
      </DialogDescription>

      <DialogBody className="space-y-6">
        <div className="rounded-md border border-gray-200 dark:border-zinc-700 bg-zinc-50 dark:bg-zinc-800 p-6">
          <div className="text-center">
            <div className="text-6xl mb-4">🚧</div>
            <div className="text-lg font-medium text-gray-900 dark:text-white mb-2">
              Coming Soon
            </div>
            <p className="text-sm text-zinc-700 dark:text-zinc-300">
              We're working on making integration creation seamless. For now, please create your {getIntegrationDisplayName()} integration through the integrations page.
            </p>
          </div>
        </div>
      </DialogBody>

      <DialogActions>
        <Button onClick={onClose}>Cancel</Button>
      </DialogActions>
    </Dialog>,
    document.body
  );
}
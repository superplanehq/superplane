import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";

export interface DeleteModalProps {
  open: boolean;
  integrationName?: string;
  canDeleteIntegrations: boolean;
  isDeleting: boolean;
  hasDeleteError: boolean;
  onDelete: () => void;
  onClose: () => void;
}

export function DeleteModal({
  open,
  integrationName,
  canDeleteIntegrations,
  isDeleting,
  hasDeleteError,
  onDelete,
  onClose,
}: DeleteModalProps) {
  if (!open) {
    return null;
  }

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50">
      <div className="bg-white dark:bg-gray-900 rounded-lg shadow-xl max-w-md w-full mx-4">
        <div className="p-6">
          <h3 className="text-lg font-semibold text-gray-800 dark:text-gray-100 mb-2">
            Delete {integrationName || "integration"}?
          </h3>
          <p className="text-sm text-gray-800 dark:text-gray-100 mb-6">
            This cannot be undone. All data will be permanently deleted.
          </p>
          <div className="flex justify-start gap-3">
            <LoadingButton
              color="blue"
              onClick={onDelete}
              disabled={!canDeleteIntegrations}
              loading={isDeleting}
              loadingText="Deleting..."
              className="bg-red-600 hover:bg-red-700 dark:bg-red-600 dark:hover:bg-red-700"
            >
              Delete
            </LoadingButton>
            <Button variant="outline" onClick={onClose} disabled={isDeleting}>
              Cancel
            </Button>
          </div>
          {hasDeleteError && (
            <div className="mt-4 p-3 bg-red-50 dark:bg-red-900/20 border border-red-200 dark:border-red-800 rounded-md">
              <p className="text-sm text-red-800 dark:text-red-200">Failed to delete integration. Please try again.</p>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

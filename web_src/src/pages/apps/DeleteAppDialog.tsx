import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/Input/input";
import { useDeleteApp } from "@/hooks/useAppData";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import type { AppsApp } from "@/lib/appsApi";

interface DeleteAppDialogProps {
  app: AppsApp;
  isOpen: boolean;
  onClose: () => void;
  redirectOnDelete?: boolean;
}

export function DeleteAppDialog({ app, isOpen, onClose, redirectOnDelete = false }: DeleteAppDialogProps) {
  const { organizationId = "" } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();
  const [confirmText, setConfirmText] = useState("");
  const deleteMutation = useDeleteApp(organizationId);

  const displayName = app.metadata?.displayName ?? "this app";
  const confirmRequired = displayName;
  const isConfirmed = confirmText === confirmRequired;

  const handleClose = () => {
    setConfirmText("");
    onClose();
  };

  const handleDelete = async () => {
    if (!app.metadata?.id || !isConfirmed) return;
    try {
      await deleteMutation.mutateAsync(app.metadata.id);
      showSuccessToast(`App "${displayName}" deleted`);
      handleClose();
      if (redirectOnDelete) {
        navigate(`/${organizationId}/apps`);
      }
    } catch {
      showErrorToast("Failed to delete app");
    }
  };

  return (
    <Dialog size="sm" open={isOpen} onClose={handleClose}>
      <DialogTitle>Delete App</DialogTitle>
      <DialogDescription>
        <p className="text-sm text-gray-700 dark:text-gray-300 mb-4">
          This will permanently delete <strong>{displayName}</strong> and its Code Storage repository. This action
          cannot be undone.
        </p>
        <p className="text-sm text-gray-700 dark:text-gray-300 mb-2">
          Type <strong>{confirmRequired}</strong> to confirm:
        </p>
        <Input
          value={confirmText}
          onChange={(e) => setConfirmText(e.target.value)}
          placeholder={confirmRequired}
          autoFocus
        />
      </DialogDescription>
      <DialogActions>
        <Button variant="outline" onClick={handleClose} disabled={deleteMutation.isPending}>
          Cancel
        </Button>
        <Button variant="destructive" onClick={handleDelete} disabled={!isConfirmed || deleteMutation.isPending}>
          {deleteMutation.isPending ? "Deleting…" : "Delete App"}
        </Button>
      </DialogActions>
    </Dialog>
  );
}

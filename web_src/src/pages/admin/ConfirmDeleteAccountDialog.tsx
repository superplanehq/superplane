import { Button } from "@/components/ui/button";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Trash2 } from "lucide-react";
import React from "react";

interface ConfirmDeleteAccountDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  accountName: string;
  accountEmail: string;
  deleting: boolean;
}

export const ConfirmDeleteAccountDialog: React.FC<ConfirmDeleteAccountDialogProps> = ({
  open,
  onClose,
  onConfirm,
  accountName,
  accountEmail,
  deleting,
}) => (
  <Dialog open={open} onClose={onClose} size="md">
    <div className="flex items-center gap-3 mb-2">
      <div className="p-2 rounded-full bg-red-100 text-red-600 dark:bg-red-950/40 dark:text-red-300">
        <Trash2 size={20} />
      </div>
      <DialogTitle className="text-gray-800 dark:text-gray-100">Delete Account</DialogTitle>
    </div>

    <DialogDescription className="text-sm text-gray-600 mt-2 space-y-2 dark:text-gray-400">
      <p>
        You are about to delete <strong>{accountName}</strong> ({accountEmail}). You cannot undo this action.
      </p>
      <ul className="list-disc pl-5 space-y-1 text-gray-500 dark:text-gray-400">
        <li>The account, its sessions, and its sign-in methods are deleted</li>
        <li>Organizations where this user is the only owner are deleted with their workspaces and automations</li>
        <li>The user can sign up again with the same email</li>
      </ul>
    </DialogDescription>

    <DialogActions>
      <Button variant="destructive" onClick={onConfirm} disabled={deleting}>
        {deleting ? "Deleting..." : "Delete Account"}
      </Button>
      <Button variant="outline" onClick={onClose} disabled={deleting}>
        Cancel
      </Button>
    </DialogActions>
  </Dialog>
);

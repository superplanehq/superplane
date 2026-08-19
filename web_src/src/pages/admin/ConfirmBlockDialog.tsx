import { Button } from "@/components/ui/button";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { Ban } from "lucide-react";
import React from "react";

interface ConfirmBlockDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  accountName: string;
  accountEmail: string;
  isBlocking: boolean;
}

export const ConfirmBlockDialog: React.FC<ConfirmBlockDialogProps> = ({
  open,
  onClose,
  onConfirm,
  accountName,
  accountEmail,
  isBlocking,
}) => (
  <Dialog open={open} onClose={onClose} size="md">
    <div className="flex items-center gap-3 mb-2">
      <div
        className={`p-2 rounded-full ${
          isBlocking
            ? "bg-red-100 text-red-600 dark:bg-red-950/40 dark:text-red-300"
            : "bg-emerald-100 text-emerald-600 dark:bg-emerald-950/40 dark:text-emerald-300"
        }`}
      >
        <Ban size={20} />
      </div>
      <DialogTitle className="text-gray-800 dark:text-gray-100">
        {isBlocking ? "Block Account" : "Unblock Account"}
      </DialogTitle>
    </div>

    <DialogDescription className="text-sm text-gray-600 mt-2 space-y-2 dark:text-gray-400">
      {isBlocking ? (
        <>
          <p>
            You are about to block <strong>{accountName}</strong> ({accountEmail}).
          </p>
          <p>They will immediately lose access to SuperPlane and see a message to contact support.</p>
          <ul className="list-disc pl-5 space-y-1 text-gray-500 dark:text-gray-400">
            <li>Existing sessions and personal API tokens are invalidated</li>
            <li>Organization API keys they created are revoked</li>
            <li>They cannot sign in until an installation admin unblocks them</li>
          </ul>
        </>
      ) : (
        <>
          <p>
            You are about to unblock <strong>{accountName}</strong> ({accountEmail}).
          </p>
          <p>They will be able to sign in again. Existing sessions stay invalid until they re-authenticate.</p>
        </>
      )}
    </DialogDescription>

    <DialogActions>
      <Button variant={isBlocking ? "destructive" : "default"} onClick={onConfirm}>
        {isBlocking ? "Block Account" : "Unblock Account"}
      </Button>
      <Button variant="outline" onClick={onClose}>
        Cancel
      </Button>
    </DialogActions>
  </Dialog>
);

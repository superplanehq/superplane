import { Button } from "@/components/ui/button";
import { Dialog, DialogActions, DialogDescription, DialogTitle } from "@/components/Dialog/dialog";
import { AlertTriangle } from "lucide-react";
import React from "react";

interface ConfirmAdminDialogProps {
  open: boolean;
  onClose: () => void;
  onConfirm: () => void;
  accountName: string;
  accountEmail: string;
  isPromoting: boolean;
}

const ConfirmAdminDialog: React.FC<ConfirmAdminDialogProps> = ({
  open,
  onClose,
  onConfirm,
  accountName,
  accountEmail,
  isPromoting,
}) => (
  <Dialog open={open} onClose={onClose} size="md">
    <div className="flex items-center gap-3 mb-2">
      <div className={`p-2 rounded-full ${isPromoting ? "bg-amber-100 text-amber-600" : "bg-red-100 text-red-600"}`}>
        <AlertTriangle size={20} />
      </div>
      <DialogTitle className="text-gray-800">
        {isPromoting ? "Promote to Installation Admin" : "Remove Installation Admin"}
      </DialogTitle>
    </div>

    <DialogDescription className="text-sm text-gray-600 mt-2 space-y-2">
      {isPromoting ? (
        <>
          <p>
            You are about to grant <strong>{accountName}</strong> ({accountEmail}) installation admin access.
          </p>
          <p>This will allow them to:</p>
          <ul className="list-disc pl-5 space-y-1 text-gray-500">
            <li>View all organizations and their data across this installation</li>
            <li>Impersonate any user in any organization</li>
            <li>Promote or demote other installation admins</li>
          </ul>
        </>
      ) : (
        <>
          <p>
            You are about to remove installation admin access from <strong>{accountName}</strong> ({accountEmail}).
          </p>
          <p>They will lose the ability to access the admin dashboard and impersonate users.</p>
        </>
      )}
    </DialogDescription>

    <DialogActions>
      <Button variant={isPromoting ? "default" : "destructive"} onClick={onConfirm}>
        {isPromoting ? "Promote to Admin" : "Remove Admin Access"}
      </Button>
      <Button variant="outline" onClick={onClose}>
        Cancel
      </Button>
    </DialogActions>
  </Dialog>
);

export default ConfirmAdminDialog;

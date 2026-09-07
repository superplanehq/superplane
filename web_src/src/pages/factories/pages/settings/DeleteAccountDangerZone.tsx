import { useState } from "react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { deleteAccount } from "@/lib/accountSettings";
import { showErrorToast } from "@/lib/toast";

import { FactorySettingsCard } from "./FactorySettingsCard";

export function DeleteAccountDangerZone({ email }: { email: string }) {
  const [open, setOpen] = useState(false);
  const [confirmation, setConfirmation] = useState("");
  const [saving, setSaving] = useState(false);
  const canDelete = confirmation.trim().toLowerCase() === email.toLowerCase();

  const handleDelete = async () => {
    if (!canDelete) {
      return;
    }
    setSaving(true);
    try {
      await deleteAccount(email);
      window.location.assign("/login");
    } catch (error) {
      showErrorToast(error instanceof Error ? error.message : "Failed to delete account.");
      setSaving(false);
    }
  };

  return (
    <>
      <FactorySettingsCard title="Danger zone" data-testid="account-redesign-danger">
        <div className="flex flex-col gap-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="min-w-0 space-y-0.5">
            <p className="text-[13px] font-medium text-foreground">Delete account</p>
            <p className="text-[12px] text-muted-foreground">
              SuperPlane deletes organizations you created. Those organizations stay for 30 days, then SuperPlane
              removes them. You lose access now.
            </p>
          </div>
          <Button
            type="button"
            size="sm"
            variant="destructive"
            onClick={() => setOpen(true)}
            data-testid="account-redesign-delete"
          >
            Delete account
          </Button>
        </div>
      </FactorySettingsCard>

      <Dialog
        open={open}
        onOpenChange={(next) => {
          setOpen(next);
          if (!next) {
            setConfirmation("");
          }
        }}
      >
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Delete account</DialogTitle>
            <DialogDescription>
              Type {email} to confirm. SuperPlane deletes organizations you created. Those organizations stay for 30
              days, then SuperPlane removes them.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="account-delete-email">Email</Label>
            <Input
              id="account-delete-email"
              value={confirmation}
              onChange={(event) => setConfirmation(event.target.value)}
              data-testid="account-redesign-delete-confirm"
            />
          </div>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setOpen(false)}>
              Keep account
            </Button>
            <LoadingButton
              variant="destructive"
              disabled={!canDelete}
              loading={saving}
              onClick={() => void handleDelete()}
              data-testid="account-redesign-delete-submit"
            >
              Delete account
            </LoadingButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

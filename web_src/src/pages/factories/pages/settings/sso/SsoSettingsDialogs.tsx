import { useId } from "react";

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
import { CopyButton } from "@/ui/CopyButton";

import type { ConfirmRequest } from "./ssoSettingsTypes";

export function ConfirmDialog({ confirm, onClose }: { confirm: ConfirmRequest | null; onClose: () => void }) {
  return (
    <Dialog open={Boolean(confirm)} onOpenChange={(open) => !open && onClose()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{confirm?.title}</DialogTitle>
          <DialogDescription>{confirm?.body}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose}>
            Cancel
          </Button>
          <Button
            type="button"
            onClick={() => {
              confirm?.run();
              onClose();
            }}
          >
            {confirm?.actionLabel || "Confirm"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function RequiredDialog({
  open,
  onOpenChange,
  onConfirm,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Require Single Sign-On</DialogTitle>
          <DialogDescription>
            Members must sign in through the identity provider to open this organization. Owners can still use another
            method when owner bypass is on.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button type="button" onClick={onConfirm}>
            Require Single Sign-On
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

export function TokenDialog({ secret, onDismiss }: { secret?: string | null; onDismiss: () => void }) {
  const id = useId();
  return (
    <Dialog open={Boolean(secret)} onOpenChange={(open) => !open && onDismiss()}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>SCIM token</DialogTitle>
          <DialogDescription>Copy this token now. You cannot see it again.</DialogDescription>
        </DialogHeader>
        <div className="flex w-full max-w-xl items-center gap-2">
          <Label className="sr-only" htmlFor={id}>
            SCIM token
          </Label>
          <Input id={id} readOnly value={secret ?? ""} className="font-mono text-[12px]" />
          <CopyButton variant="icon" text={secret ?? ""} ariaLabel="Copy SCIM token" />
        </div>
      </DialogContent>
    </Dialog>
  );
}

import { useState } from "react";
import { Github } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";

import { FactorySettingsCard } from "../FactorySettingsCard";
import { SettingsActionRow } from "./accountProfileRedesignParts";

export function AccountProfileVelocityGithubCard({
  username,
  onLink,
  onRemove,
}: {
  username: string | null;
  onLink: () => void;
  onRemove: () => void;
}) {
  const [removeOpen, setRemoveOpen] = useState(false);

  return (
    <>
      <FactorySettingsCard title="GitHub for Velocity" data-testid="account-redesign-velocity-github">
        <SettingsActionRow
          title={
            <span className="inline-flex items-center gap-2">
              <Github className="size-4" aria-hidden />
              {username ? `Linked as ${username}` : "Not linked"}
            </span>
          }
          description="Velocity uses this link to credit pull requests. This link does not change how you sign in."
          action={
            username ? (
              <Button type="button" size="sm" variant="ghost" onClick={() => setRemoveOpen(true)}>
                Remove
              </Button>
            ) : (
              <Button type="button" size="sm" variant="outline" onClick={onLink}>
                Link GitHub
              </Button>
            )
          }
        />
      </FactorySettingsCard>
      <RemoveVelocityGithubDialog
        open={removeOpen}
        onOpenChange={setRemoveOpen}
        onConfirm={() => {
          onRemove();
          setRemoveOpen(false);
        }}
      />
    </>
  );
}

function RemoveVelocityGithubDialog({
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
          <DialogTitle>Remove the GitHub link</DialogTitle>
          <DialogDescription>
            Velocity reports stop crediting your pull requests to you. Your sign-in methods do not change.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Keep the link
          </Button>
          <Button type="button" variant="destructive" onClick={onConfirm}>
            Remove link
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

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

export function AccountProfileAssociatedAccountsCard({
  githubUsername,
  onLinkGithub,
  onRemoveGithub,
}: {
  githubUsername: string | null;
  onLinkGithub: () => void;
  onRemoveGithub: () => void;
}) {
  const [removeOpen, setRemoveOpen] = useState(false);

  return (
    <>
      <FactorySettingsCard title="Associated accounts" data-testid="account-redesign-associated-accounts">
        <p className="text-[12px] text-muted-foreground">
          SuperPlane uses these accounts to credit your work. This does not change how you sign in.
        </p>
        <ul className="mt-4 space-y-4">
          <li>
            <SettingsActionRow
              title={
                <span className="inline-flex items-center gap-2">
                  <Github className="size-4" aria-hidden />
                  GitHub
                </span>
              }
              description={
                githubUsername
                  ? `Linked as ${githubUsername}. Velocity uses this GitHub account to credit your pull requests.`
                  : "Velocity uses this GitHub account to credit your pull requests."
              }
              testId="account-redesign-associated-github"
              action={
                githubUsername ? (
                  <Button type="button" size="sm" variant="ghost" onClick={() => setRemoveOpen(true)}>
                    Remove
                  </Button>
                ) : (
                  <Button type="button" size="sm" variant="outline" onClick={onLinkGithub}>
                    Link GitHub
                  </Button>
                )
              }
            />
          </li>
        </ul>
      </FactorySettingsCard>
      <RemoveAssociatedGithubDialog
        open={removeOpen}
        onOpenChange={setRemoveOpen}
        onConfirm={() => {
          onRemoveGithub();
          setRemoveOpen(false);
        }}
      />
    </>
  );
}

function RemoveAssociatedGithubDialog({
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

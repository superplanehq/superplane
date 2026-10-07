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
  githubAccounts,
  onLinkGithub,
  onRemoveGithub,
}: {
  githubAccounts: Array<{ providerId: string; username: string }>;
  onLinkGithub: () => void;
  onRemoveGithub: (providerId: string) => void;
}) {
  const [accountToRemove, setAccountToRemove] = useState<{ providerId: string; username: string } | null>(null);

  return (
    <>
      <FactorySettingsCard title="Associated accounts" data-testid="account-redesign-associated-accounts">
        <p className="text-[12px] text-muted-foreground">
          SuperPlane uses these accounts to credit your work. This does not change how you sign in.
        </p>
        <ul className="mt-4 space-y-4">
          {githubAccounts.map((account, index) => (
            <li key={account.providerId}>
              <SettingsActionRow
                title={
                  <span className="inline-flex items-center gap-2">
                    <Github className="size-4" aria-hidden />
                    GitHub
                  </span>
                }
                description={`Linked as ${account.username}. Velocity uses this GitHub account to credit your pull requests.`}
                testId={
                  index === 0
                    ? "account-redesign-associated-github"
                    : `account-redesign-associated-github-${account.providerId}`
                }
                action={
                  <Button type="button" size="sm" variant="ghost" onClick={() => setAccountToRemove(account)}>
                    Remove
                  </Button>
                }
              />
            </li>
          ))}
          {githubAccounts.length === 0 ? (
            <li>
              <SettingsActionRow
                title={
                  <span className="inline-flex items-center gap-2">
                    <Github className="size-4" aria-hidden />
                    GitHub
                  </span>
                }
                description="Velocity uses this GitHub account to credit your pull requests."
                testId="account-redesign-associated-github"
                action={
                  <Button type="button" size="sm" variant="outline" onClick={onLinkGithub}>
                    Link GitHub
                  </Button>
                }
              />
            </li>
          ) : null}
        </ul>
        {githubAccounts.length > 0 ? (
          <Button type="button" size="sm" variant="outline" className="mt-4" onClick={onLinkGithub}>
            Link another GitHub account
          </Button>
        ) : null}
      </FactorySettingsCard>
      <RemoveAssociatedGithubDialog
        username={accountToRemove?.username ?? ""}
        open={accountToRemove !== null}
        onOpenChange={(open) => {
          if (!open) setAccountToRemove(null);
        }}
        onConfirm={() => {
          if (accountToRemove) onRemoveGithub(accountToRemove.providerId);
          setAccountToRemove(null);
        }}
      />
    </>
  );
}

function RemoveAssociatedGithubDialog({
  username,
  open,
  onOpenChange,
  onConfirm,
}: {
  username: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Remove {username}?</DialogTitle>
          <DialogDescription>
            Velocity reports will no longer credit pull requests from this GitHub account. Your sign-in methods do not
            change.
          </DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={() => onOpenChange(false)}>
            Keep account
          </Button>
          <Button type="button" variant="destructive" onClick={onConfirm}>
            Remove account
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

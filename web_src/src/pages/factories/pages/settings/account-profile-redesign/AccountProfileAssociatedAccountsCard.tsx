import { useState, type ReactNode } from "react";
import { Github } from "lucide-react";

import bitbucketIcon from "@/assets/icons/integrations/bitbucket.svg";
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

type LinkedIdentity = { providerId: string; username: string };

type AssociatedProvider = {
  key: "github" | "bitbucket";
  title: string;
  testId: string;
  icon: ReactNode;
  accounts: LinkedIdentity[];
  emptyDescription: string;
  linkedDescription: (username: string) => string;
  linkLabel: string;
  linkAnotherLabel: string;
  removeDescription: string;
  onLink: () => void;
  onRemove: (providerId: string) => void;
};

export function AccountProfileAssociatedAccountsCard({
  githubAccounts,
  bitbucketAccounts,
  onLinkGithub,
  onLinkBitbucket,
  onRemoveGithub,
  onRemoveBitbucket,
}: {
  githubAccounts: LinkedIdentity[];
  bitbucketAccounts: LinkedIdentity[];
  onLinkGithub: () => void;
  onLinkBitbucket: () => void;
  onRemoveGithub: (providerId: string) => void;
  onRemoveBitbucket: (providerId: string) => void;
}) {
  const [accountToRemove, setAccountToRemove] = useState<AccountToRemove | null>(null);
  const providers = associatedProviders({
    githubAccounts,
    bitbucketAccounts,
    onLinkGithub,
    onLinkBitbucket,
    onRemoveGithub,
    onRemoveBitbucket,
  });

  return (
    <>
      <FactorySettingsCard title="Associated accounts" data-testid="account-redesign-associated-accounts">
        <p className="text-[12px] text-muted-foreground">
          SuperPlane uses these accounts to credit your work. This does not change how you sign in.
        </p>
        <AssociatedProviderList providers={providers} onRemove={setAccountToRemove} />
      </FactorySettingsCard>
      <RemoveAssociatedAccountDialog
        username={accountToRemove?.username ?? ""}
        description={accountToRemove?.description ?? ""}
        open={accountToRemove !== null}
        onOpenChange={(open) => {
          if (!open) setAccountToRemove(null);
        }}
        onConfirm={() => {
          if (accountToRemove) accountToRemove.onRemove(accountToRemove.providerId);
          setAccountToRemove(null);
        }}
      />
    </>
  );
}

type AccountToRemove = {
  providerId: string;
  username: string;
  description: string;
  onRemove: (providerId: string) => void;
};

function associatedProviders({
  githubAccounts,
  bitbucketAccounts,
  onLinkGithub,
  onLinkBitbucket,
  onRemoveGithub,
  onRemoveBitbucket,
}: {
  githubAccounts: LinkedIdentity[];
  bitbucketAccounts: LinkedIdentity[];
  onLinkGithub: () => void;
  onLinkBitbucket: () => void;
  onRemoveGithub: (providerId: string) => void;
  onRemoveBitbucket: (providerId: string) => void;
}): AssociatedProvider[] {
  return [
    {
      key: "github",
      title: "GitHub",
      testId: "account-redesign-associated-github",
      icon: <Github className="size-4" aria-hidden />,
      accounts: githubAccounts,
      emptyDescription: "Velocity uses this GitHub account to credit your pull requests.",
      linkedDescription: (username) =>
        `Linked as ${username}. Velocity uses this GitHub account to credit your pull requests.`,
      linkLabel: "Link GitHub",
      linkAnotherLabel: "Link another GitHub account",
      removeDescription:
        "Velocity reports will no longer credit pull requests from this GitHub account. Your sign-in methods do not change.",
      onLink: onLinkGithub,
      onRemove: onRemoveGithub,
    },
    {
      key: "bitbucket",
      title: "Bitbucket",
      testId: "account-redesign-associated-bitbucket",
      icon: <img src={bitbucketIcon} alt="" className="size-4" />,
      accounts: bitbucketAccounts,
      emptyDescription: "This link does not change how you sign in.",
      linkedDescription: (username) => `Linked as ${username}. This link does not change how you sign in.`,
      linkLabel: "Link Bitbucket",
      linkAnotherLabel: "Link another Bitbucket account",
      removeDescription: "SuperPlane removes this Bitbucket link. Your sign-in methods do not change.",
      onLink: onLinkBitbucket,
      onRemove: onRemoveBitbucket,
    },
  ];
}

function AssociatedProviderList({
  providers,
  onRemove,
}: {
  providers: AssociatedProvider[];
  onRemove: (account: AccountToRemove) => void;
}) {
  return (
    <div className="mt-4 space-y-4">
      {providers.map((provider) => (
        <div key={provider.key}>
          <ul className="space-y-4">
            {provider.accounts.map((account, index) => (
              <li key={account.providerId}>
                <SettingsActionRow
                  title={<ProviderTitle icon={provider.icon} title={provider.title} />}
                  description={provider.linkedDescription(account.username)}
                  testId={index === 0 ? provider.testId : `${provider.testId}-${account.providerId}`}
                  action={
                    <Button
                      type="button"
                      size="sm"
                      variant="ghost"
                      onClick={() =>
                        onRemove({
                          providerId: account.providerId,
                          username: account.username,
                          description: provider.removeDescription,
                          onRemove: provider.onRemove,
                        })
                      }
                    >
                      Remove
                    </Button>
                  }
                />
              </li>
            ))}
            {provider.accounts.length === 0 ? (
              <li>
                <SettingsActionRow
                  title={<ProviderTitle icon={provider.icon} title={provider.title} />}
                  description={provider.emptyDescription}
                  testId={provider.testId}
                  action={
                    <Button type="button" size="sm" variant="outline" onClick={provider.onLink}>
                      {provider.linkLabel}
                    </Button>
                  }
                />
              </li>
            ) : null}
          </ul>
          {provider.accounts.length > 0 ? (
            <Button type="button" size="sm" variant="outline" className="mt-4" onClick={provider.onLink}>
              {provider.linkAnotherLabel}
            </Button>
          ) : null}
        </div>
      ))}
    </div>
  );
}

function ProviderTitle({ icon, title }: { icon: ReactNode; title: string }) {
  return (
    <span className="inline-flex items-center gap-2">
      {icon}
      {title}
    </span>
  );
}

function RemoveAssociatedAccountDialog({
  username,
  description,
  open,
  onOpenChange,
  onConfirm,
}: {
  username: string;
  description: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onConfirm: () => void;
}) {
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Remove {username}?</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
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

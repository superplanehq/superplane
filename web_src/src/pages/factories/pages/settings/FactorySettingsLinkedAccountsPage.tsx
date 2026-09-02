import { useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router";

import { useAccount } from "@/contexts/useAccount";
import { disconnectLinkedAccount, linkedAccountConnectHref } from "@/lib/accountSettings";
import { getApiErrorMessage } from "@/lib/errors";
import { showErrorToast, showSuccessToast } from "@/lib/toast";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Github } from "lucide-react";

import { FactorySettingsCard, FactorySettingsPageFrame } from "./FactorySettingsCard";
import { SettingsActionRow } from "./account-profile-redesign/accountProfileRedesignParts";

function useConsumeLinkResult(refreshAccount: () => Promise<void>) {
  const location = useLocation();
  const navigate = useNavigate();

  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const error = params.get("auth_error");
    const result = params.get("linked_account");
    if (!error && !result) {
      return;
    }
    if (error === "linked_account_in_use") {
      showErrorToast("Another SuperPlane account already uses this GitHub account.");
    }
    if (result === "linked") {
      showSuccessToast("GitHub account linked.");
      void refreshAccount();
    }
    params.delete("auth_error");
    params.delete("linked_account");
    params.delete("provider");
    const search = params.toString();
    void navigate({ pathname: location.pathname, search: search ? `?${search}` : "" }, { replace: true });
  }, [location.pathname, location.search, navigate, refreshAccount]);

  return location;
}

export function FactorySettingsLinkedAccountsPage() {
  const { account, refreshAccount } = useAccount();
  const [removeOpen, setRemoveOpen] = useState(false);
  const location = useConsumeLinkResult(refreshAccount);

  if (!account) {
    return <p className="text-[13px] text-muted-foreground">Loading linked accounts…</p>;
  }

  const github = (account.linked_accounts ?? []).find((linked) => linked.provider === "github");

  const remove = () => {
    setRemoveOpen(false);
    void disconnectLinkedAccount("github")
      .then(async () => {
        await refreshAccount();
        showSuccessToast("GitHub link removed.");
      })
      .catch((error) => {
        showErrorToast(getApiErrorMessage(error, "Failed to remove the linked account."));
      });
  };

  return (
    <>
      <FactorySettingsPageFrame
        title="Linked accounts"
        subtitle="Tell SuperPlane who you are on the services your team uses."
      >
        <FactorySettingsCard title="GitHub" data-testid="linked-accounts-github-card">
          <p className="text-[12px] text-muted-foreground">
            Velocity reports use this link to credit the pull requests you author. A linked account does not change how
            you sign in.
          </p>
          <div className="mt-4">
            <SettingsActionRow
              title={
                <span className="inline-flex items-center gap-2">
                  <Github className="size-4" aria-hidden="true" />
                  GitHub
                </span>
              }
              description={github ? `Linked as ${github.username}.` : "Not linked."}
              testId="linked-accounts-github"
              action={
                github ? (
                  <Button type="button" size="sm" variant="ghost" onClick={() => setRemoveOpen(true)}>
                    Remove
                  </Button>
                ) : (
                  <Button
                    type="button"
                    size="sm"
                    variant="outline"
                    onClick={() => {
                      window.location.assign(
                        linkedAccountConnectHref("github", `${location.pathname}${location.search}`),
                      );
                    }}
                  >
                    Link GitHub
                  </Button>
                )
              }
            />
          </div>
        </FactorySettingsCard>
      </FactorySettingsPageFrame>

      <Dialog open={removeOpen} onOpenChange={setRemoveOpen}>
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Remove the GitHub link</DialogTitle>
            <DialogDescription>
              Velocity reports stop crediting your pull requests to you. Your sign-in methods do not change.
            </DialogDescription>
          </DialogHeader>
          <DialogFooter>
            <Button type="button" variant="outline" onClick={() => setRemoveOpen(false)}>
              Keep the link
            </Button>
            <Button type="button" variant="destructive" onClick={remove}>
              Remove link
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

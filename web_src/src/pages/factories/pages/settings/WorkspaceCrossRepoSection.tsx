import { useState } from "react";
import { X } from "lucide-react";

import { PermissionTooltip } from "@/components/PermissionGate";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";

import { FirstRunMissingAccessLine } from "../onboarding/first-run/FirstRunMissingAccessLine";
import { IntegrationChoiceIcon, RepositoryPicker } from "../onboarding/onboardingSteps";
import type { VcsHostId } from "../onboarding/onboardingFixtures";
import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { FactorySettingsCard } from "./FactorySettingsCard";

/** Visual only. Chosen repositories stay in this list and are not saved. */
export function CrossRepoContextCard({
  canUpdate,
  host,
  bitbucket = false,
  connected,
  loading = false,
  error = false,
  catalog,
  initial = [],
  granting = false,
  onGrantAccess,
}: {
  canUpdate: boolean;
  host: VcsHostId;
  bitbucket?: boolean;
  connected: boolean;
  loading?: boolean;
  error?: boolean;
  catalog: string[];
  initial?: string[];
  granting?: boolean;
  onGrantAccess?: () => void;
}) {
  const [added, setAdded] = useState(initial);
  const [adding, setAdding] = useState(false);
  const available = catalog.filter((name) => !added.includes(name));
  const grantAction = bitbucket
    ? AGENT_RESOURCES_COPY.crossRepoGrantBitbucket
    : AGENT_RESOURCES_COPY.crossRepoGrantGitHub;

  const choose = (name: string) => {
    setAdded((current) => (current.includes(name) ? current : [...current, name]));
    setAdding(false);
  };

  return (
    <>
      <FactorySettingsCard
        title={AGENT_RESOURCES_COPY.crossRepoTitle}
        description={AGENT_RESOURCES_COPY.crossRepoHelper}
        action={
          <PermissionTooltip allowed={canUpdate} message={AGENT_RESOURCES_COPY.noUpdatePermission}>
            <Button
              type="button"
              size="sm"
              disabled={!canUpdate}
              onClick={() => setAdding(true)}
              data-testid="agent-cross-repo-add"
            >
              {AGENT_RESOURCES_COPY.addRepository}
            </Button>
          </PermissionTooltip>
        }
        data-testid="agent-cross-repo"
      >
        {added.length > 0 ? (
          <ul
            className="divide-y divide-border overflow-hidden rounded-lg border border-border"
            data-testid="agent-cross-repo-list"
          >
            {added.map((name) => (
              <li key={name} className="flex min-w-0 items-center gap-3 px-3 py-2.5">
                <IntegrationChoiceIcon name={host} />
                <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{name}</span>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="size-8 shrink-0 text-muted-foreground"
                  disabled={!canUpdate}
                  aria-label={AGENT_RESOURCES_COPY.removeRepository}
                  onClick={() => setAdded((current) => current.filter((entry) => entry !== name))}
                >
                  <X className="size-4" strokeWidth={1.75} aria-hidden />
                </Button>
              </li>
            ))}
          </ul>
        ) : null}
      </FactorySettingsCard>
      <Dialog open={adding} onOpenChange={setAdding}>
        <DialogContent data-testid="agent-cross-repo-picker">
          <DialogHeader>
            <DialogTitle>{AGENT_RESOURCES_COPY.addRepository}</DialogTitle>
            <DialogDescription>{AGENT_RESOURCES_COPY.crossRepoDialogHelper}</DialogDescription>
          </DialogHeader>
          <RepositoryChoice
            host={host}
            bitbucket={bitbucket}
            connected={connected}
            loading={loading}
            error={error}
            catalog={available}
            disabled={!canUpdate}
            onSelect={choose}
          />
          {connected && !loading ? (
            <FirstRunMissingAccessLine
              hint={AGENT_RESOURCES_COPY.crossRepoMissingHint}
              question={AGENT_RESOURCES_COPY.crossRepoMissingQuestion}
              action={grantAction}
              disabled={!canUpdate || !onGrantAccess}
              loading={granting}
              loadingText={
                bitbucket ? AGENT_RESOURCES_COPY.crossRepoOpeningBitbucket : AGENT_RESOURCES_COPY.crossRepoOpeningGitHub
              }
              onClick={() => onGrantAccess?.()}
            />
          ) : null}
        </DialogContent>
      </Dialog>
    </>
  );
}

function RepositoryChoice({
  host,
  bitbucket,
  connected,
  loading,
  error,
  catalog,
  disabled,
  onSelect,
}: {
  host: VcsHostId;
  bitbucket: boolean;
  connected: boolean;
  loading: boolean;
  error: boolean;
  catalog: string[];
  disabled: boolean;
  onSelect: (name: string) => void;
}) {
  if (!connected) {
    return (
      <p className="text-[13px] text-muted-foreground">
        {bitbucket ? AGENT_RESOURCES_COPY.crossRepoConnectBitbucket : AGENT_RESOURCES_COPY.crossRepoConnectGitHub}
      </p>
    );
  }
  if (loading) {
    return <p className="text-[13px] text-muted-foreground">{AGENT_RESOURCES_COPY.crossRepoLoading}</p>;
  }
  return (
    <>
      <RepositoryPicker host={host} repos={catalog} selectedRepo={null} onSelect={onSelect} disabled={disabled} />
      {error ? <p className="mt-3 text-[13px] text-destructive">{AGENT_RESOURCES_COPY.crossRepoLoadError}</p> : null}
    </>
  );
}

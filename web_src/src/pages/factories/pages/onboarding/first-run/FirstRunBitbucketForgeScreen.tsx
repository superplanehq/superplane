import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { useState } from "react";

import { RepositoryPicker } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.bitbucket;
const chooseCopy = FIRST_RUN_COPY.choose;

type FirstRunBitbucketForgeScreenProps = {
  phase: "connect" | "grant" | "choose";
  connectHref: string;
  installUrl: string;
  repositories: string[];
  selectedRepository: string | null;
  granting?: boolean;
  saving?: boolean;
  loading?: boolean;
  loadError?: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onGrantAccess: () => void;
  onSelectRepository: (repository: string) => void;
  onContinue: () => void;
};

export function FirstRunBitbucketForgeScreen({
  phase,
  connectHref,
  installUrl,
  repositories,
  selectedRepository,
  granting = false,
  saving = false,
  loading = false,
  loadError = false,
  chrome,
  sphere,
  onGrantAccess,
  onSelectRepository,
  onContinue,
}: FirstRunBitbucketForgeScreenProps) {
  if (loading) {
    return (
      <FirstRunShell testId="first-run-bitbucket-connect" chrome={chrome} sphere={sphere}>
        <p className="text-[13px] text-muted-foreground" role="status">
          {copy.loading}
        </p>
      </FirstRunShell>
    );
  }

  if (phase === "connect") {
    return (
      <FirstRunShell testId="first-run-bitbucket-connect" chrome={chrome} sphere={sphere}>
        <FirstRunHeading headline={copy.connectHeadline}>
          <p className="text-[15px] leading-6 text-muted-foreground">{copy.connectAccountBody}</p>
        </FirstRunHeading>
        <div className="mt-8">
          <Button asChild>
            <a href={connectHref} data-testid="first-run-bitbucket-oauth">
              {copy.connectAction}
            </a>
          </Button>
        </div>
      </FirstRunShell>
    );
  }

  if (phase === "grant") {
    return (
      <FirstRunShell testId="first-run-bitbucket-grant" chrome={chrome} sphere={sphere} busy={granting}>
        <FirstRunHeading headline={copy.connectHeadline}>
          <p className="text-[15px] leading-6 text-muted-foreground">{copy.grantBody}</p>
        </FirstRunHeading>
        <div className="mt-8 space-y-4">
          <LoadingButton
            type="button"
            onClick={onGrantAccess}
            loading={granting}
            loadingText={copy.openingBitbucket}
            data-testid="first-run-bitbucket-install"
          >
            {copy.grantAction}
          </LoadingButton>
          {installUrl ? <InstallationLink url={installUrl} /> : null}
          {loadError ? <p className="text-[13px] text-destructive">{copy.loadError}</p> : null}
        </div>
      </FirstRunShell>
    );
  }

  const busy = saving;
  return (
    <FirstRunShell
      testId="first-run-bitbucket-choose"
      chrome={chrome}
      busy={busy}
      contentSpacing="compact"
      sphere={sphere}
    >
      <FirstRunHeading headline={chooseCopy.headline}>
        <p className="text-[13px] text-muted-foreground">{copy.repositoryHelper}</p>
      </FirstRunHeading>
      <div className="mt-8 space-y-4">
        {loadError ? <p className="text-[13px] text-destructive">{copy.loadError}</p> : null}
        {repositories.length === 0 ? (
          <p className="text-[13px] text-muted-foreground">{copy.empty}</p>
        ) : (
          <RepositoryPicker
            host="bitbucket"
            repos={repositories}
            selectedRepo={selectedRepository}
            disabled={busy}
            listClassName="max-h-48"
            onSelect={onSelectRepository}
          />
        )}
        <div className="space-y-2">
          <LoadingButton
            type="button"
            className="w-full"
            disabled={!selectedRepository || busy}
            loading={saving}
            loadingText={chooseCopy.saving}
            onClick={onContinue}
            data-testid="first-run-continue-to-tickets"
          >
            {selectedRepository ? chooseCopy.continueReady : chooseCopy.continue}
          </LoadingButton>
          <p className="text-[12px] text-muted-foreground">{chooseCopy.moreLater}</p>
        </div>
      </div>
    </FirstRunShell>
  );
}

function InstallationLink({ url }: { url: string }) {
  const [copied, setCopied] = useState(false);
  return (
    <div className="space-y-2">
      <Label htmlFor="bitbucket-install-link">{copy.grantLinkLabel}</Label>
      <div className="flex gap-2">
        <Input id="bitbucket-install-link" readOnly value={url} data-testid="first-run-bitbucket-install-link" />
        <Button
          type="button"
          variant="outline"
          onClick={() => {
            void navigator.clipboard.writeText(url).then(() => {
              setCopied(true);
            });
          }}
        >
          {copied ? copy.copied : copy.copyLink}
        </Button>
      </div>
    </div>
  );
}

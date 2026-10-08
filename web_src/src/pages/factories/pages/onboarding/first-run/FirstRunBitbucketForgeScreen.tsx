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
  installedWorkspaces?: string[];
  selectedRepository: string | null;
  granting?: boolean;
  saving?: boolean;
  loading?: boolean;
  loadError?: boolean;
  lookupFailed?: boolean;
  retrying?: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onRetryLookup?: () => void;
  onGrantAccess: () => void;
  onSelectRepository: (repository: string) => void;
  onContinue: () => void;
};

export function FirstRunBitbucketForgeScreen(props: FirstRunBitbucketForgeScreenProps) {
  if (props.loading) return <BitbucketLoadingScreen {...props} />;
  if (props.lookupFailed) return <BitbucketLookupFailedScreen {...props} />;
  if (props.phase === "connect") return <BitbucketConnectScreen {...props} />;
  if (props.phase === "grant") return <BitbucketGrantScreen {...props} />;
  return <BitbucketChooseRepositoryScreen {...props} />;
}

function BitbucketLoadingScreen({ chrome, sphere }: FirstRunBitbucketForgeScreenProps) {
  return (
    <FirstRunShell testId="first-run-bitbucket-connect" chrome={chrome} sphere={sphere} visual="preview">
      <p className="text-[13px] text-muted-foreground" role="status">
        {copy.loading}
      </p>
    </FirstRunShell>
  );
}

function BitbucketLookupFailedScreen({
  retrying = false,
  chrome,
  sphere,
  onRetryLookup = () => undefined,
}: FirstRunBitbucketForgeScreenProps) {
  return (
    <FirstRunShell testId="first-run-bitbucket-lookup-failed" chrome={chrome} sphere={sphere} visual="preview">
      <FirstRunHeading headline={copy.connectHeadline}>
        <p className="text-[13px] text-destructive">{copy.lookupFailed}</p>
      </FirstRunHeading>
      <div className="mt-8">
        <LoadingButton
          type="button"
          onClick={onRetryLookup}
          loading={retrying}
          loadingText={copy.retrying}
          data-testid="first-run-bitbucket-retry"
        >
          {copy.retry}
        </LoadingButton>
      </div>
    </FirstRunShell>
  );
}

function BitbucketConnectScreen({ connectHref, chrome, sphere }: FirstRunBitbucketForgeScreenProps) {
  return (
    <FirstRunShell testId="first-run-bitbucket-connect" chrome={chrome} sphere={sphere} visual="preview">
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

function BitbucketGrantScreen({
  installUrl,
  granting = false,
  loadError = false,
  chrome,
  sphere,
  onGrantAccess,
}: FirstRunBitbucketForgeScreenProps) {
  return (
    <FirstRunShell testId="first-run-bitbucket-grant" chrome={chrome} sphere={sphere} busy={granting} visual="preview">
      <FirstRunHeading headline={copy.connectHeadline}>
        <p className="text-[15px] leading-6 text-muted-foreground">{copy.grantBody}</p>
      </FirstRunHeading>
      <div className="mt-8 space-y-4">
        <p className="text-[13px] text-muted-foreground" role="status">
          {copy.synchronizing}
        </p>
        <p className="text-[13px] text-muted-foreground">{copy.waitingForInstallation}</p>
        <p className="text-[13px] text-muted-foreground">{copy.installIfNeeded}</p>
        <LoadingButton
          type="button"
          variant="outline"
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

function BitbucketChooseRepositoryScreen({
  repositories,
  installedWorkspaces = [],
  selectedRepository,
  saving = false,
  loadError = false,
  chrome,
  sphere,
  onSelectRepository,
  onContinue,
  onRetryLookup,
  onGrantAccess,
  retrying = false,
  granting = false,
}: FirstRunBitbucketForgeScreenProps) {
  return (
    <FirstRunShell
      testId="first-run-bitbucket-choose"
      chrome={chrome}
      busy={saving}
      contentSpacing="compact"
      sphere={sphere}
      visual="preview"
    >
      <FirstRunHeading headline={chooseCopy.headline}>
        <p className="text-[13px] text-muted-foreground">{copy.repositoryHelper}</p>
      </FirstRunHeading>
      <div className="mt-8 space-y-4">
        {loadError ? <p className="text-[13px] text-destructive">{copy.loadError}</p> : null}
        {repositories.length === 0 ? (
          <div className="space-y-3">
            <p className="text-[13px]">{copy.installed}</p>
            <ul className="text-[13px] text-muted-foreground">
              {installedWorkspaces.map((workspace) => (
                <li key={workspace}>{workspace}</li>
              ))}
            </ul>
            <p className="text-[13px] text-muted-foreground">{copy.installedEmpty}</p>
            <LoadingButton type="button" onClick={onRetryLookup} loading={retrying} loadingText={copy.retrying}>
              {copy.checkRepositories}
            </LoadingButton>
            <LoadingButton
              type="button"
              variant="outline"
              onClick={onGrantAccess}
              loading={granting}
              loadingText={copy.openingBitbucket}
            >
              {copy.installAnother}
            </LoadingButton>
          </div>
        ) : (
          <RepositoryPicker
            host="bitbucket"
            repos={repositories}
            selectedRepo={selectedRepository}
            disabled={saving}
            listClassName="max-h-48"
            onSelect={onSelectRepository}
          />
        )}
        <div className="space-y-2">
          <LoadingButton
            type="button"
            className="w-full"
            disabled={!selectedRepository || saving}
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

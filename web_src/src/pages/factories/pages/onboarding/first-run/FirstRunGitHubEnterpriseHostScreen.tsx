import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useState, type ReactNode } from "react";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";
import type { GitHubEnterpriseHostPhase } from "./githubEnterpriseHostPhase";

const copy = FIRST_RUN_COPY.githubEnterprise;

function looksLikeGitHubHost(value: string): boolean {
  const host = value.trim().toLowerCase();
  if (!host || host.includes("/") || host.includes(" ")) {
    return false;
  }
  return host.includes(".");
}

function looksLikeGitHubOwner(value: string): boolean {
  const owner = value.trim();
  return Boolean(owner) && !owner.includes("/") && !owner.includes(" ");
}

function GitHubEnterpriseShell({
  chrome,
  sphere,
  headline,
  body,
  children,
}: {
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  headline: string;
  body: string;
  children: ReactNode;
}) {
  return (
    <FirstRunShell testId="first-run-github-enterprise-host" chrome={chrome} sphere={sphere} visual="preview">
      <FirstRunHeading headline={headline}>
        <p className="text-[15px] leading-6 text-muted-foreground">{body}</p>
      </FirstRunHeading>
      {children}
    </FirstRunShell>
  );
}

function ConnectError({ message }: { message?: string }) {
  if (!message) return null;
  return (
    <p className="text-[13px] text-destructive" role="alert">
      {message}
    </p>
  );
}

function GitHubEnterpriseAppPhase({
  connecting,
  chrome,
  sphere,
  onResumeSetup,
}: {
  connecting: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onResumeSetup?: () => void;
}) {
  return (
    <GitHubEnterpriseShell chrome={chrome} sphere={sphere} headline={copy.finishHeadline} body={copy.finishBody}>
      <LoadingButton
        type="button"
        className="mt-8 w-full"
        loading={connecting}
        data-testid="first-run-github-enterprise-finish"
        onClick={() => onResumeSetup?.()}
      >
        {connecting ? copy.opening : copy.finishAction}
      </LoadingButton>
    </GitHubEnterpriseShell>
  );
}

function GitHubEnterpriseOrganizationPhase({
  owner,
  connecting,
  connectError,
  chrome,
  sphere,
  onOwnerChange,
  onContinueOrganization,
}: {
  owner: string;
  connecting: boolean;
  connectError?: string;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onOwnerChange: (owner: string) => void;
  onContinueOrganization: (owner: string) => void;
}) {
  const trimmedOwner = owner.trim();
  const valid = looksLikeGitHubOwner(trimmedOwner);
  return (
    <GitHubEnterpriseShell
      chrome={chrome}
      sphere={sphere}
      headline={copy.organizationHeadline}
      body={copy.organizationBody}
    >
      <form
        className="mt-8 space-y-6"
        onSubmit={(event) => {
          event.preventDefault();
          if (!valid || connecting) return;
          onContinueOrganization(trimmedOwner);
        }}
      >
        <div className="space-y-2">
          <Label htmlFor="first-run-github-enterprise-organization">{copy.organizationLabel}</Label>
          <Input
            id="first-run-github-enterprise-organization"
            value={owner}
            onChange={(event) => onOwnerChange(event.target.value)}
            placeholder={copy.organizationPlaceholder}
            autoComplete="off"
            spellCheck={false}
            disabled={connecting}
            data-testid="first-run-github-enterprise-organization"
          />
        </div>
        <ConnectError message={connectError} />
        <LoadingButton
          type="submit"
          className="w-full"
          loading={connecting}
          disabled={!valid}
          data-testid="first-run-github-enterprise-create-app"
        >
          {connecting ? copy.opening : copy.organizationContinue}
        </LoadingButton>
      </form>
    </GitHubEnterpriseShell>
  );
}

function GitHubEnterpriseHostnamePhase({
  host,
  connecting,
  connectError,
  chrome,
  sphere,
  onHostChange,
  onContinueHost,
}: {
  host: string;
  connecting: boolean;
  connectError?: string;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onHostChange: (host: string) => void;
  onContinueHost: (host: string) => void;
}) {
  const trimmedHost = host.trim();
  const valid = looksLikeGitHubHost(trimmedHost);
  return (
    <GitHubEnterpriseShell chrome={chrome} sphere={sphere} headline={copy.headline} body={copy.body}>
      <form
        className="mt-8 space-y-6"
        onSubmit={(event) => {
          event.preventDefault();
          if (!valid || connecting) return;
          onContinueHost(trimmedHost);
        }}
      >
        <div className="space-y-2">
          <Label htmlFor="first-run-github-enterprise-hostname">{copy.hostLabel}</Label>
          <Input
            id="first-run-github-enterprise-hostname"
            value={host}
            onChange={(event) => onHostChange(event.target.value)}
            placeholder={copy.hostPlaceholder}
            autoComplete="off"
            spellCheck={false}
            disabled={connecting}
            data-testid="first-run-github-enterprise-hostname"
          />
          <p className="text-[13px] leading-5 text-muted-foreground">
            {copy.ipAllowList}{" "}
            <a
              href={copy.ipAllowListHref}
              target="_blank"
              rel="noopener noreferrer"
              className="underline underline-offset-2"
            >
              {copy.ipAllowListLink}
            </a>
            .
          </p>
        </div>
        <ConnectError message={connectError} />
        <LoadingButton
          type="submit"
          className="w-full"
          loading={connecting}
          disabled={!valid}
          data-testid="first-run-github-enterprise-continue"
        >
          {connecting ? copy.checking : copy.continue}
        </LoadingButton>
      </form>
    </GitHubEnterpriseShell>
  );
}

export function FirstRunGitHubEnterpriseHostScreen({
  initialHost = "",
  connecting = false,
  phase = "host",
  connectError,
  chrome,
  sphere,
  onContinueHost,
  onContinueOrganization,
  onResumeSetup,
}: {
  initialHost?: string;
  connecting?: boolean;
  phase?: GitHubEnterpriseHostPhase;
  connectError?: string;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onContinueHost: (host: string) => void;
  onContinueOrganization: (owner: string) => void;
  onResumeSetup?: () => void;
}) {
  const [host, setHost] = useState(initialHost);
  const [owner, setOwner] = useState("");
  if (phase === "app") {
    return (
      <GitHubEnterpriseAppPhase connecting={connecting} chrome={chrome} sphere={sphere} onResumeSetup={onResumeSetup} />
    );
  }
  if (phase === "organization") {
    return (
      <GitHubEnterpriseOrganizationPhase
        owner={owner}
        connecting={connecting}
        connectError={connectError}
        chrome={chrome}
        sphere={sphere}
        onOwnerChange={setOwner}
        onContinueOrganization={onContinueOrganization}
      />
    );
  }
  return (
    <GitHubEnterpriseHostnamePhase
      host={host}
      connecting={connecting}
      connectError={connectError}
      chrome={chrome}
      sphere={sphere}
      onHostChange={setHost}
      onContinueHost={onContinueHost}
    />
  );
}

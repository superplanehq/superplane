import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useState } from "react";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.githubEnterprise;

export type GitHubEnterpriseHostPhase = "host" | "organization" | "app";

export function githubEnterpriseHostPhase(args: {
  integrationId?: string;
  ready?: boolean;
  currentStepName?: string;
  showHost?: boolean;
}): GitHubEnterpriseHostPhase {
  if (args.showHost || args.ready) return "host";
  if (args.currentStepName === "createApp") return "app";
  if (args.integrationId?.trim()) return "organization";
  return "host";
}

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
  const trimmedHost = host.trim();
  const trimmedOwner = owner.trim();

  if (phase === "app") {
    return (
      <FirstRunShell testId="first-run-github-enterprise-host" chrome={chrome} sphere={sphere} visual="preview">
        <FirstRunHeading headline={copy.finishHeadline}>
          <p className="text-[15px] leading-6 text-muted-foreground">{copy.finishBody}</p>
        </FirstRunHeading>
        <LoadingButton
          type="button"
          className="mt-8 w-full"
          loading={connecting}
          data-testid="first-run-github-enterprise-finish"
          onClick={() => onResumeSetup?.()}
        >
          {connecting ? copy.opening : copy.finishAction}
        </LoadingButton>
      </FirstRunShell>
    );
  }

  if (phase === "organization") {
    const valid = looksLikeGitHubOwner(trimmedOwner);
    return (
      <FirstRunShell testId="first-run-github-enterprise-host" chrome={chrome} sphere={sphere} visual="preview">
        <FirstRunHeading headline={copy.organizationHeadline}>
          <p className="text-[15px] leading-6 text-muted-foreground">{copy.organizationBody}</p>
        </FirstRunHeading>
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
              onChange={(event) => setOwner(event.target.value)}
              placeholder={copy.organizationPlaceholder}
              autoComplete="off"
              spellCheck={false}
              disabled={connecting}
              data-testid="first-run-github-enterprise-organization"
            />
          </div>
          {connectError ? (
            <p className="text-[13px] text-destructive" role="alert">
              {connectError}
            </p>
          ) : null}
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
      </FirstRunShell>
    );
  }

  const valid = looksLikeGitHubHost(trimmedHost);
  return (
    <FirstRunShell testId="first-run-github-enterprise-host" chrome={chrome} sphere={sphere} visual="preview">
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[15px] leading-6 text-muted-foreground">{copy.body}</p>
      </FirstRunHeading>

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
            onChange={(event) => setHost(event.target.value)}
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
        {connectError ? (
          <p className="text-[13px] text-destructive" role="alert">
            {connectError}
          </p>
        ) : null}
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
    </FirstRunShell>
  );
}

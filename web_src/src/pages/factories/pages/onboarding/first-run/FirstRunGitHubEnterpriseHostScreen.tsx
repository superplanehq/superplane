import { LoadingButton } from "@/components/ui/loading-button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { useState } from "react";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.githubEnterprise;

function looksLikeGitHubHost(value: string): boolean {
  const host = value.trim().toLowerCase();
  if (!host || host.includes("/") || host.includes(" ")) {
    return false;
  }
  return host.includes(".");
}

export function FirstRunGitHubEnterpriseHostScreen({
  initialHost = "",
  connecting = false,
  chrome,
  sphere,
  onContinue,
}: {
  initialHost?: string;
  connecting?: boolean;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  onContinue: (host: string) => void;
}) {
  const [host, setHost] = useState(initialHost);
  const trimmed = host.trim();
  const valid = looksLikeGitHubHost(trimmed);

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
          onContinue(trimmed);
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
        <LoadingButton
          type="submit"
          className="w-full"
          loading={connecting}
          disabled={!valid}
          data-testid="first-run-github-enterprise-continue"
        >
          {connecting ? copy.opening : copy.continue}
        </LoadingButton>
      </form>
    </FirstRunShell>
  );
}

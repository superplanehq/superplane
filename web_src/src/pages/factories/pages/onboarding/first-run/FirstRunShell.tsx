import { cn } from "@/lib/utils";
import type { ReactNode } from "react";

import { SidebarUserMenu } from "../../../layout/SidebarUserMenu";
import { useFactoriesThemeClass } from "../../../lib/useFactoriesThemeClass";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import type { FirstRunChrome } from "./firstRunTypes";

export const FIRST_RUN_STEP_COUNT = 5;

export function FirstRunShell({
  children,
  testId,
  chrome,
  width = "narrow",
}: {
  children: ReactNode;
  testId: string;
  chrome?: FirstRunChrome;
  width?: "narrow" | "wide";
}) {
  useFactoriesThemeClass();
  const copy = FIRST_RUN_COPY.chrome;
  const stepCount = chrome?.stepCount ?? FIRST_RUN_STEP_COUNT;

  return (
    <div className="fixed inset-0 bg-background text-foreground" data-testid={testId}>
      {chrome ? (
        <div className="absolute bottom-4 left-4 z-10">
          <SidebarUserMenu
            variant="floating"
            organizationId={chrome.organizationId}
            organizationName={chrome.organizationName}
            factoryKey={chrome.factoryKey}
            userName={chrome.displayName || chrome.email || "You"}
            userEmail={chrome.email}
            userAvatarUrl={chrome.avatarUrl}
            onQuitOnboarding={chrome.onQuitOnboarding}
            defaultOpen={chrome.menuDefaultOpen}
          />
        </div>
      ) : null}

      <div className="flex h-full items-center justify-center overflow-y-auto px-6 py-24">
        <div className={cn("w-full text-center", width === "wide" ? "max-w-xl" : "max-w-md")}>{children}</div>
      </div>

      {chrome ? (
        <nav
          className="pointer-events-none absolute inset-x-0 bottom-0 z-10 flex justify-center pb-6"
          aria-label={copy.stepLabel(chrome.stepIndex + 1, stepCount)}
        >
          <ol className="flex items-center gap-1.5">
            {Array.from({ length: stepCount }, (_, index) => (
              <li
                key={index}
                className={cn(
                  "size-1.5 rounded-full",
                  index === chrome.stepIndex ? "bg-foreground" : "bg-muted-foreground/30",
                )}
                aria-current={index === chrome.stepIndex ? "step" : undefined}
              />
            ))}
          </ol>
        </nav>
      ) : null}
    </div>
  );
}

export function FirstRunHeading({
  greeting,
  headline,
  size = "page",
  children,
}: {
  greeting?: string;
  headline: string;
  size?: "page" | "display";
  children?: ReactNode;
}) {
  return (
    <header className="mx-auto max-w-lg space-y-3">
      {greeting ? <p className="text-[15px] font-medium tracking-[-0.01em]">{greeting}</p> : null}
      <h1
        className={cn(
          size === "display"
            ? "text-[26px] font-semibold leading-8 tracking-[-0.03em]"
            : "workspace-page-title font-semibold",
        )}
      >
        {headline}
      </h1>
      {children}
    </header>
  );
}

export function FirstRunPanel({ children }: { children: ReactNode }) {
  return <div className="rounded-lg border border-border bg-card p-4 text-left">{children}</div>;
}

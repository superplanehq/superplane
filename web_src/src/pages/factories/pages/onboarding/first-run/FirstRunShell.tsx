import { OrganizationSwitchMenu } from "@/components/OrganizationSwitchMenu";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { ArrowRightLeft } from "lucide-react";
import type { ReactNode } from "react";

import { useFactoriesThemeClass } from "../../../lib/useFactoriesThemeClass";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunSpherePane, type FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";
import { FirstRunWorkspaceSwitch } from "./FirstRunWorkspaceSwitch";

export const FIRST_RUN_STEP_COUNT = 5;

export function FirstRunShell({
  children,
  testId,
  chrome,
  busy = false,
  width = "narrow",
  sphere,
}: {
  children: ReactNode;
  testId: string;
  chrome?: FirstRunChrome;
  busy?: boolean;
  width?: "narrow" | "wide";
  sphere?: FirstRunSphereProps;
}) {
  useFactoriesThemeClass();
  const controlsDisabled = busy || Boolean(chrome?.busy);

  return (
    <div
      className="fixed inset-0 bg-background text-foreground"
      data-testid={testId}
      aria-busy={controlsDisabled || undefined}
    >
      <FirstRunTopBar chrome={chrome} disabled={controlsDisabled} />

      {sphere ? (
        <div className="flex h-full">
          <div className="flex flex-1 items-center overflow-y-auto px-8 py-24 lg:px-12">
            <div className="w-full max-w-lg text-left">
              {children}
              <FirstRunBack onBack={chrome?.onBack} disabled={controlsDisabled} />
            </div>
          </div>
          <FirstRunSpherePane {...sphere} />
        </div>
      ) : (
        <div className="flex h-full items-center justify-center overflow-y-auto px-6 py-24">
          <div className={cn("w-full text-center", width === "wide" ? "max-w-xl" : "max-w-md")}>
            {children}
            <FirstRunBack onBack={chrome?.onBack} disabled={controlsDisabled} />
          </div>
        </div>
      )}

      <FirstRunWorkspaceSwitch switcher={chrome?.workspaceSwitch} disabled={controlsDisabled} />
      <FirstRunProgress chrome={chrome} />
    </div>
  );
}

function FirstRunTopBar({ chrome, disabled }: { chrome?: FirstRunChrome; disabled: boolean }) {
  const identity = chrome?.email ?? chrome?.displayName;
  return (
    <div className="pointer-events-none absolute inset-x-0 top-0 z-10 flex items-start justify-between px-6 py-5">
      <div className="pointer-events-auto flex items-center gap-3">
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className="text-muted-foreground hover:text-foreground"
          onClick={chrome?.onLogOut}
          disabled={disabled}
          data-testid="first-run-log-out"
        >
          {FIRST_RUN_COPY.chrome.logOut}
        </Button>
        <FirstRunOrganizationSwitch organizationSwitch={chrome?.organizationSwitch} disabled={disabled} />
      </div>
      {identity ? (
        <p className="text-right text-[13px] leading-5 text-muted-foreground" data-testid="first-run-signed-in">
          <span className="block">{FIRST_RUN_COPY.chrome.loggedInAs}</span>
          <span className="block text-foreground">{identity}</span>
        </p>
      ) : null}
    </div>
  );
}

function FirstRunBack({ onBack, disabled }: { onBack?: () => void; disabled: boolean }) {
  if (!onBack) return null;
  return (
    <Button
      type="button"
      variant="ghost"
      className="mt-4 text-muted-foreground hover:text-foreground"
      onClick={onBack}
      disabled={disabled}
      data-testid="first-run-back"
    >
      {FIRST_RUN_COPY.chrome.back}
    </Button>
  );
}

function FirstRunProgress({ chrome }: { chrome?: FirstRunChrome }) {
  if (!chrome) return null;
  const stepCount = chrome.stepCount ?? FIRST_RUN_STEP_COUNT;
  return (
    <nav
      className="pointer-events-none absolute inset-x-0 bottom-0 z-10 flex justify-center pb-6"
      aria-label={FIRST_RUN_COPY.chrome.stepLabel(chrome.stepIndex + 1, stepCount)}
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
  );
}

function FirstRunOrganizationSwitch({
  organizationSwitch,
  disabled,
}: {
  organizationSwitch: FirstRunChrome["organizationSwitch"];
  disabled?: boolean;
}) {
  if (!organizationSwitch) return null;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="icon-xs"
          className="text-muted-foreground hover:text-foreground"
          aria-label={FIRST_RUN_COPY.chrome.switchOrganization}
          disabled={disabled}
          data-testid="first-run-organization-switch"
        >
          <ArrowRightLeft className="size-4" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-64">
        <OrganizationSwitchMenu
          currentOrganizationRouteId={organizationSwitch.currentOrganizationRouteId}
          navigateToCurrentOrganization
          testIdPrefix="first-run"
        />
      </DropdownMenuContent>
    </DropdownMenu>
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
    <header className="max-w-lg space-y-3">
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

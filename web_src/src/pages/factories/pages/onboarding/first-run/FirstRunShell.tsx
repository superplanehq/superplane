import { OrganizationSwitchMenu } from "@/components/OrganizationSwitchMenu";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { DropdownMenu, DropdownMenuContent, DropdownMenuTrigger } from "@/ui/dropdownMenu";
import { ArrowRightLeft } from "lucide-react";
import { createContext, useContext, type CSSProperties, type ReactNode } from "react";

import wordmark from "@/assets/superplane-wordmark.svg";

import { useFactoriesThemeClass } from "../../../lib/useFactoriesThemeClass";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunSpherePane, type FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome } from "./firstRunTypes";
import { FirstRunWorkspaceSwitch } from "./FirstRunWorkspaceSwitch";

export const FIRST_RUN_STEP_COUNT = 5;

export type FirstRunVisual = "app" | "preview";

const FirstRunVisualContext = createContext<FirstRunVisual>("app");

const PREVIEW_TOKENS = {
  "--background": "#11110e",
  "--foreground": "#eeede9",
  "--card": "#11110e",
  "--card-foreground": "#eeede9",
  "--primary": "#f3f3f1",
  "--primary-foreground": "#11110e",
  "--muted": "#201f1a",
  "--muted-foreground": "#9b9993",
  "--accent": "#201f1a",
  "--accent-foreground": "#eeede9",
  "--border": "#34322b",
  "--input": "#34322b",
  "--field": "#201f1a",
  "--popover": "#201f1a",
  "--popover-foreground": "#eeede9",
  "--destructive": "#e5484d",
  "--ring": "#9b9993",
} as CSSProperties;

const PREVIEW_CONTENT_CLASS = cn(
  "w-full max-w-[528px] text-left",
  "[&_[data-slot=button].bg-primary]:h-auto",
  "[&_[data-slot=button].bg-primary]:rounded-[5px]",
  "[&_[data-slot=button].bg-primary]:border-0",
  "[&_[data-slot=button].bg-primary]:px-4",
  "[&_[data-slot=button].bg-primary]:shadow-none",
  "[&_[data-slot=button].bg-primary]:hover:bg-white",
  "[&_[data-slot=button].bg-primary[data-size=default]]:py-[13px]",
  "[&_[data-slot=button].bg-primary[data-size=default]]:font-mono",
  "[&_[data-slot=button].bg-primary[data-size=default]]:text-[14px]",
  "[&_[data-slot=button].bg-primary[data-size=default]]:uppercase",
  "[&_[data-slot=button].bg-primary[data-size=default]]:tracking-[0.02em]",
  "[&_[data-slot=button].bg-primary[data-size=sm]]:py-[9px]",
  "[&_[data-slot=button].bg-primary[data-size=sm]]:font-inter",
  "[&_[data-slot=button].bg-primary[data-size=sm]]:text-[14px]",
  "[&_[data-slot=button].bg-primary[data-size=sm]]:font-normal",
  "[&_[data-slot=button].bg-primary[data-size=sm]]:normal-case",
  "[&_[data-slot=input]]:rounded-[5px]",
  "[&_[data-slot=input]]:border-[#34322b]!",
  "[&_[data-slot=input]]:bg-[#201f1a]!",
  "[&_[data-slot=input]]:text-[#eeede9]!",
  "[&_[data-slot=input]]:placeholder:text-[#9b9993]!",
);

function firstRunSide(aside: ReactNode | undefined, sphere: FirstRunSphereProps | undefined) {
  if (aside) return aside;
  if (!sphere) return null;
  return <FirstRunSpherePane {...sphere} />;
}

export function FirstRunShell({
  children,
  testId,
  chrome,
  busy = false,
  width = "narrow",
  contentSpacing = "default",
  sphere,
  aside,
  visual = "app",
}: {
  children: ReactNode;
  testId: string;
  chrome?: FirstRunChrome;
  busy?: boolean;
  width?: "narrow" | "wide";
  contentSpacing?: "default" | "compact";
  sphere?: FirstRunSphereProps;
  aside?: ReactNode;
  visual?: FirstRunVisual;
}) {
  useFactoriesThemeClass();
  const controlsDisabled = busy || Boolean(chrome?.busy);
  const side = firstRunSide(aside, sphere);

  if (visual === "preview") {
    return (
      <FirstRunVisualContext.Provider value="preview">
        <PreviewShell
          testId={testId}
          chrome={chrome}
          controlsDisabled={controlsDisabled}
          contentSpacing={contentSpacing}
          side={side}
          background={sphere?.art?.background}
        >
          {children}
        </PreviewShell>
      </FirstRunVisualContext.Provider>
    );
  }

  return (
    <AppShell
      testId={testId}
      chrome={chrome}
      controlsDisabled={controlsDisabled}
      contentSpacing={contentSpacing}
      width={width}
      side={side}
    >
      {children}
    </AppShell>
  );
}

function AppShell({
  children,
  testId,
  chrome,
  controlsDisabled,
  contentSpacing,
  width,
  side,
}: {
  children: ReactNode;
  testId: string;
  chrome?: FirstRunChrome;
  controlsDisabled: boolean;
  contentSpacing: "default" | "compact";
  width: "narrow" | "wide";
  side: ReactNode;
}) {
  return (
    <FirstRunVisualContext.Provider value="app">
      <div
        className="fixed inset-0 bg-background text-foreground"
        data-testid={testId}
        data-visual="app"
        aria-busy={controlsDisabled || undefined}
      >
        <FirstRunTopBar chrome={chrome} disabled={controlsDisabled} />

        {side ? (
          <div className="flex h-full">
            <div
              className={cn(
                "flex min-h-0 flex-1 items-center overflow-y-auto px-8 lg:px-12",
                contentSpacing === "compact" ? "py-16" : "py-24",
              )}
              data-testid="first-run-content"
            >
              <div className="mx-auto w-full max-w-lg text-left">
                {children}
                <FirstRunBack onBack={chrome?.onBack} disabled={controlsDisabled} />
              </div>
            </div>
            {side}
          </div>
        ) : (
          <div
            className={cn(
              "flex h-full min-h-0 items-center justify-center overflow-y-auto px-6",
              contentSpacing === "compact" ? "py-16" : "py-24",
            )}
            data-testid="first-run-content"
          >
            <div className={cn("w-full text-center", width === "wide" ? "max-w-xl" : "max-w-md")}>
              {children}
              <FirstRunBack onBack={chrome?.onBack} disabled={controlsDisabled} />
            </div>
          </div>
        )}

        <FirstRunWorkspaceSwitch switcher={chrome?.workspaceSwitch} disabled={controlsDisabled} />
        <FirstRunProgress chrome={chrome} />
      </div>
    </FirstRunVisualContext.Provider>
  );
}

function PreviewShell({
  children,
  testId,
  chrome,
  controlsDisabled,
  contentSpacing,
  side,
  background,
}: {
  children: ReactNode;
  testId: string;
  chrome?: FirstRunChrome;
  controlsDisabled: boolean;
  contentSpacing: "default" | "compact";
  side: ReactNode;
  background?: string;
}) {
  return (
    <div
      className="fixed inset-0 flex flex-col overflow-auto bg-[#11110e] font-inter text-[#eeede9] min-[860px]:flex-row min-[860px]:overflow-hidden"
      style={PREVIEW_TOKENS}
      data-testid={testId}
      data-visual="preview"
      aria-busy={controlsDisabled || undefined}
    >
      <div className="flex min-w-0 flex-1 flex-col px-5 py-6 min-[860px]:min-h-0 min-[860px]:px-11 min-[860px]:py-8">
        <img src={wordmark} alt="SuperPlane" className="h-[21px] w-auto self-start" data-testid="first-run-logo" />
        <div
          className={cn(
            "flex min-h-0 flex-1 items-center overflow-y-auto",
            contentSpacing === "compact" ? "py-16" : "py-8",
          )}
          data-testid="first-run-content"
        >
          <div className={PREVIEW_CONTENT_CLASS}>
            {children}
            <FirstRunBack onBack={chrome?.onBack} disabled={controlsDisabled} preview />
          </div>
        </div>
        <div className="flex items-center justify-between pt-4" data-testid="first-run-bottom-row">
          <div className="flex items-center gap-3">
            <FirstRunLogOut onLogOut={chrome?.onLogOut} disabled={controlsDisabled} preview />
            <FirstRunOrganizationSwitch organizationSwitch={chrome?.organizationSwitch} disabled={controlsDisabled} />
            <FirstRunWorkspaceSwitch switcher={chrome?.workspaceSwitch} disabled={controlsDisabled} inline />
          </div>
          <FirstRunProgress chrome={chrome} preview />
        </div>
      </div>
      {side ? (
        <>
          <div className="hidden w-px shrink-0 bg-[#34322b] min-[860px]:block" data-testid="first-run-divider" />
          <div
            className="relative h-[60vh] shrink-0 min-[860px]:h-auto min-[860px]:min-h-0 min-[860px]:flex-1"
            style={{ backgroundColor: background }}
            data-testid="first-run-art-stage"
          >
            {side}
            <FirstRunSignedIn chrome={chrome} preview />
          </div>
        </>
      ) : null}
    </div>
  );
}

function FirstRunTopBar({ chrome, disabled }: { chrome?: FirstRunChrome; disabled: boolean }) {
  return (
    <div className="pointer-events-none absolute inset-x-0 top-0 z-10 flex items-start justify-between px-6 py-5">
      <div className="pointer-events-auto flex items-center gap-3">
        <FirstRunLogOut onLogOut={chrome?.onLogOut} disabled={disabled} />
        <FirstRunOrganizationSwitch organizationSwitch={chrome?.organizationSwitch} disabled={disabled} />
      </div>
      <FirstRunSignedIn chrome={chrome} />
    </div>
  );
}

function FirstRunSignedIn({ chrome, preview = false }: { chrome?: FirstRunChrome; preview?: boolean }) {
  const identity = chrome?.email ?? chrome?.displayName;
  if (!identity) return null;
  return (
    <p
      className={cn(
        "text-right text-[14px] leading-[1.3]",
        preview
          ? "absolute top-[22px] right-7 z-10 flex flex-col items-end gap-[5px]"
          : "text-[13px] leading-5 text-muted-foreground",
      )}
      data-testid="first-run-signed-in"
    >
      <span className={cn("block", preview && "text-[rgba(238,237,233,0.5)]")}>{FIRST_RUN_COPY.chrome.loggedInAs}</span>
      <span className={cn("block", preview ? "text-[#eeede9]" : "text-foreground")}>{identity}</span>
    </p>
  );
}

function FirstRunLogOut({
  onLogOut,
  disabled,
  preview = false,
}: {
  onLogOut?: () => void;
  disabled: boolean;
  preview?: boolean;
}) {
  if (!onLogOut) return null;
  return (
    <Button
      type="button"
      variant="ghost"
      size="sm"
      className={cn(
        preview
          ? "h-auto px-0 text-[14px] font-normal text-[#9b9993] hover:bg-transparent hover:text-[#eeede9]"
          : "text-muted-foreground hover:text-foreground",
      )}
      onClick={onLogOut}
      disabled={disabled}
      data-testid="first-run-log-out"
    >
      {FIRST_RUN_COPY.chrome.logOut}
    </Button>
  );
}

function FirstRunBack({
  onBack,
  disabled,
  preview = false,
}: {
  onBack?: () => void;
  disabled: boolean;
  preview?: boolean;
}) {
  if (!onBack) return null;
  return (
    <Button
      type="button"
      variant="ghost"
      className={cn(
        "mt-4",
        preview
          ? "h-auto px-0 text-[14px] font-normal text-[#9b9993] hover:bg-transparent hover:text-[#eeede9]"
          : "text-muted-foreground hover:text-foreground",
      )}
      onClick={onBack}
      disabled={disabled}
      data-testid="first-run-back"
    >
      {FIRST_RUN_COPY.chrome.back}
    </Button>
  );
}

function FirstRunProgress({ chrome, preview = false }: { chrome?: FirstRunChrome; preview?: boolean }) {
  if (!chrome) return null;
  const stepCount = chrome.stepCount ?? FIRST_RUN_STEP_COUNT;
  return (
    <nav
      className={cn(
        "z-10",
        preview ? undefined : "pointer-events-none absolute inset-x-0 bottom-0 flex justify-center pb-6",
      )}
      aria-label={FIRST_RUN_COPY.chrome.stepLabel(chrome.stepIndex + 1, stepCount)}
      data-testid="first-run-progress"
    >
      <ol className="flex items-center gap-1.5">
        {Array.from({ length: stepCount }, (_, index) => (
          <li
            key={index}
            className={cn(
              "size-1.5 rounded-full",
              preview
                ? index === chrome.stepIndex
                  ? "bg-[#eeede9]"
                  : "bg-[#34322b]"
                : index === chrome.stepIndex
                  ? "bg-foreground"
                  : "bg-muted-foreground/30",
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
  const preview = useContext(FirstRunVisualContext) === "preview";
  return (
    <header className={cn("space-y-3", preview ? "max-w-[528px]" : "max-w-lg")}>
      {greeting ? (
        <p
          className={preview ? "text-[16px] font-normal text-[#eeede9]" : "text-[15px] font-medium tracking-[-0.01em]"}
        >
          {greeting}
        </p>
      ) : null}
      <h1
        className={cn(
          preview
            ? "text-[24px] font-normal leading-[1.3] text-[#eeede9]"
            : size === "display"
              ? "text-[26px] font-semibold leading-8 tracking-[-0.03em]"
              : "workspace-page-title font-semibold",
        )}
      >
        {headline}
      </h1>
      {children ? (
        <div
          className={
            preview
              ? "space-y-3 text-[16px] leading-[1.45] text-[#9b9993] [&_p]:text-[16px] [&_p]:leading-[1.45] [&_p]:text-[#9b9993]"
              : undefined
          }
        >
          {children}
        </div>
      ) : null}
    </header>
  );
}

export function FirstRunPanel({ children }: { children: ReactNode }) {
  return <div className="rounded-lg border border-border bg-card p-4 text-left">{children}</div>;
}

import { Button } from "@/components/ui/button";
import { RequirePermission } from "@/components/PermissionGate";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { cn } from "@/lib/utils";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { ArrowLeft, ArrowRight, Eye, Plus } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { AppDetailModal, LeadIcon, type AppEntry } from "../AppDetailModal";
import { APP_CATALOG } from "../appCatalog";
import { HomePageShell } from "../HomePageShell";
import {
  homeCardTitleClassName,
  homeListCardClassName,
  homePageSubtitleClassName,
  homePageTitleClassName,
} from "../homePageStyles";
import { InstallProgressPanel } from "../InstallProgressPanel";
import { useCreateApp } from "../useCreateApp";

type SetupTool = {
  label: string;
  integrationName?: string;
  iconSlug?: string;
};

const SETUP_STEPS: {
  title: string;
  detail: string;
  tools: SetupTool[];
}[] = [
  {
    title: "Trigger",
    detail: "Manual prompt, issue from a tracker, and/or PR tag.",
    tools: [
      { label: "GitHub Issues", integrationName: "github" },
      { label: "GitLab Issues", integrationName: "gitlab" },
      { label: "Linear", integrationName: "linear" },
      { label: "Jira", integrationName: "jira" },
    ],
  },
  {
    title: "Version control",
    detail: "Connect GitHub or GitLab for checkout and PR/MR.",
    tools: [
      { label: "GitHub", integrationName: "github" },
      { label: "GitLab", integrationName: "gitlab" },
    ],
  },
  {
    title: "Coding agent",
    detail: "Harness + API key. Agents run in our sandbox.",
    tools: [
      { label: "Claude Code", integrationName: "claude" },
      { label: "Codex", integrationName: "openai" },
      { label: "Open Code", integrationName: "opencode" },
    ],
  },
  {
    title: "Preview and tweak",
    detail: "Adjust prompts, SSH commands, and other settings.",
    tools: [],
  },
];

const OUTCOME_STEPS: {
  title: string;
  detail: string;
  phase: "ready" | "running" | "review" | "done";
}[] = [
  {
    title: "Work is triggered",
    detail: "Manual prompt, issue, or PR tag starts a run.",
    phase: "ready",
  },
  {
    title: "Agent plans and codes",
    detail: "Harness runs in the SuperPlane sandbox.",
    phase: "running",
  },
  {
    title: "Opens a pull request",
    detail: "Branch + PR or merge request.",
    phase: "running",
  },
  {
    title: "Keeps checks passing",
    detail: "Watches PR checks and loops on failures until they pass.",
    phase: "running",
  },
  {
    title: "Waits for your review",
    detail: "You stay on the loop.",
    phase: "review",
  },
  {
    title: "Addresses review comments",
    detail: "Agent updates the PR from feedback.",
    phase: "running",
  },
  {
    title: "Gets to a mergeable state",
    detail: "PR ready for you to merge.",
    phase: "done",
  },
];

/**
 * Storybook-only fresh-org landing POC: densified split (decision + outcome)
 * with quiet escape hatches for blank apps and the starter catalog.
 * Not mounted in production routes.
 */
export function FreshOrgLandingPage() {
  return (
    <RequirePermission resource="canvases" action="create">
      <HomePageShell>
        <FreshOrgLandingPoc />
      </HomePageShell>
    </RequirePermission>
  );
}

export function FreshOrgLandingPoc() {
  const { createApp, isSaving } = useCreateApp();
  const [showCatalog, setShowCatalog] = useState(false);
  const [showFactoryStub, setShowFactoryStub] = useState(false);
  const [visibleCount, setVisibleCount] = useState(7);
  const [selectedApp, setSelectedApp] = useState<AppEntry | null>(null);
  const [installingApp, setInstallingApp] = useState<AppEntry | null>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const busy = isSaving || installingApp !== null;
  const visible = APP_CATALOG.slice(0, visibleCount);

  useEffect(() => {
    if (!showCatalog) return;
    const el = sentinelRef.current;
    if (!el) return;
    if (typeof IntersectionObserver === "undefined") {
      setVisibleCount(APP_CATALOG.length);
      return;
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setVisibleCount((prev) => Math.min(prev + 7, APP_CATALOG.length));
        }
      },
      { rootMargin: "100px" },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [showCatalog]);

  if (showFactoryStub) {
    return (
      <div className="mx-auto w-full max-w-3xl px-8 py-16">
        <button
          type="button"
          onClick={() => setShowFactoryStub(false)}
          className="mb-8 inline-flex items-center gap-1.5 text-sm font-medium text-gray-500 hover:text-slate-900 dark:text-gray-400 dark:hover:text-gray-100"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden />
          Back
        </button>
        <h1 className={homePageTitleClassName}>Software Factory setup</h1>
        <p className={homePageSubtitleClassName}>
          Next screens will walk through each step. This Storybook view is a placeholder for that flow.
        </p>
        <ol className="mt-8 space-y-4">
          {SETUP_STEPS.map((step, index) => (
            <li
              key={step.title}
              className={cn(
                "flex gap-4 rounded-lg bg-white px-4 py-3 outline outline-slate-950/10",
                "dark:bg-gray-900 dark:outline-gray-700/70",
              )}
            >
              <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-slate-100 text-xs font-semibold text-slate-700 dark:bg-gray-800 dark:text-gray-200">
                {index + 1}
              </span>
              <div className="min-w-0">
                <p className={homeCardTitleClassName}>{step.title}</p>
                <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">{step.detail}</p>
                {step.tools.length > 0 && <ToolChipRow tools={step.tools} />}
                <p className="mt-2 text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
                  Next
                </p>
              </div>
            </li>
          ))}
        </ol>
      </div>
    );
  }

  return (
    <>
      {selectedApp && (
        <AppDetailModal
          app={selectedApp}
          busy={busy}
          onBack={() => setSelectedApp(null)}
          onInstall={(e) => {
            e.stopPropagation();
            setInstallingApp(selectedApp);
            setSelectedApp(null);
          }}
          onClose={() => setSelectedApp(null)}
        />
      )}

      <div className="mx-auto w-full max-w-6xl px-8 py-14 lg:py-20">
        <div className="grid items-start gap-12 lg:grid-cols-[0.9fr_1.1fr] lg:gap-14">
          <div>
            <p className="text-[11px] font-semibold uppercase tracking-[0.06em] text-gray-400 dark:text-gray-500">
              Recommended
            </p>
            <h1 className="mt-3 max-w-[16ch] text-2xl font-semibold tracking-tight text-slate-900 sm:text-3xl dark:text-gray-100">
              Ship PRs to a mergeable state
            </h1>
            <p className={cn(homePageSubtitleClassName, "mt-3 max-w-md")}>
              An automated app that orchestrates cloud agents to solve issues end to end. Agents run in the SuperPlane
              sandbox. You review; the factory keeps going until the PR is mergeable.
            </p>
            <div className="mt-7">
              <Button type="button" size="lg" onClick={() => setShowFactoryStub(true)}>
                Start setup
                <ArrowRight />
              </Button>
            </div>

            <ol className="mt-8 space-y-4" aria-label="Setup steps">
              {SETUP_STEPS.map((step, index) => (
                <li key={step.title} className="flex gap-3">
                  <span className="mt-0.5 flex h-6 w-6 shrink-0 items-center justify-center rounded-full bg-slate-900 text-[11px] font-semibold text-white dark:bg-gray-100 dark:text-slate-900">
                    {index + 1}
                  </span>
                  <div className="min-w-0">
                    <p className="text-sm font-semibold text-slate-900 dark:text-gray-100">{step.title}</p>
                    <p className="mt-0.5 text-sm leading-snug text-gray-500 dark:text-gray-400">{step.detail}</p>
                  </div>
                </li>
              ))}
            </ol>

            <div className="mt-8 flex flex-wrap items-center gap-x-3 gap-y-2 text-sm text-gray-400 dark:text-gray-500">
              <span>Or</span>
              <button
                type="button"
                disabled={busy}
                onClick={() => {
                  if (busy) return;
                  void createApp(generateCanvasName());
                }}
                className="inline-flex items-center gap-1.5 font-medium text-gray-500 underline-offset-4 hover:text-slate-900 hover:underline disabled:opacity-50 dark:text-gray-400 dark:hover:text-gray-100"
              >
                <Plus className="h-3.5 w-3.5" aria-hidden />
                Create a blank app
              </button>
              <span className="text-slate-300 dark:text-gray-600" aria-hidden>
                ·
              </span>
              <button
                type="button"
                onClick={() => setShowCatalog((open) => !open)}
                className="font-medium text-gray-500 underline-offset-4 hover:text-slate-900 hover:underline dark:text-gray-400 dark:hover:text-gray-100"
                aria-expanded={showCatalog}
              >
                {showCatalog ? "Hide starter apps" : "Browse other starter apps"}
              </button>
            </div>
          </div>

          <aside
            className={cn(
              "rounded-xl bg-white px-6 py-6 outline outline-slate-950/10",
              "dark:bg-gray-900 dark:outline-gray-700/60",
            )}
          >
            <h2 className="text-sm font-semibold text-slate-900 dark:text-gray-100">What you get</h2>
            <p className="mt-1 text-sm text-gray-500 dark:text-gray-400">
              End-to-end orchestration from trigger to a mergeable PR.
            </p>
            <ol className="relative mt-6 space-y-0">
              {OUTCOME_STEPS.map((step, index) => {
                const isLast = index === OUTCOME_STEPS.length - 1;
                return (
                  <li key={step.title} className="relative flex gap-4 pb-6 last:pb-0">
                    {!isLast && (
                      <span
                        className="absolute top-9 bottom-0 left-[15px] w-px bg-slate-200 dark:bg-gray-700"
                        aria-hidden
                      />
                    )}
                    <span
                      className={cn(
                        "relative z-10 flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs font-bold",
                        step.phase === "ready" && "bg-sky-100 text-sky-700 dark:bg-sky-950/40 dark:text-sky-300",
                        step.phase === "running" &&
                          "bg-emerald-100 text-emerald-700 dark:bg-emerald-950/50 dark:text-emerald-300",
                        step.phase === "review" &&
                          "bg-orange-100 text-orange-700 dark:bg-orange-950/40 dark:text-orange-300",
                        step.phase === "done" && "bg-sky-100 text-sky-700 dark:bg-sky-950/40 dark:text-sky-300",
                      )}
                    >
                      {index + 1}
                    </span>
                    <div className="min-w-0 pt-0.5">
                      <p className="text-sm font-semibold text-slate-900 dark:text-gray-100">{step.title}</p>
                      <p className="mt-0.5 text-sm leading-snug text-gray-500 dark:text-gray-400">{step.detail}</p>
                    </div>
                  </li>
                );
              })}
            </ol>
          </aside>
        </div>

        {showCatalog && (
          <div className="mt-10 flex flex-col gap-3">
            <p className="text-center text-xs font-medium text-gray-400 dark:text-gray-500">
              Automation starters (not Software Factory setup)
            </p>
            {visible.map((app) => (
              <StarterAppListItem
                key={app.repo}
                app={app}
                busy={busy}
                isInstalling={installingApp?.repo === app.repo}
                onSelect={setSelectedApp}
                onInstall={(entry) => {
                  if (busy) return;
                  setInstallingApp(entry);
                  setSelectedApp(null);
                }}
                onCloseInstall={() => setInstallingApp(null)}
              />
            ))}
            {visibleCount < APP_CATALOG.length && <div ref={sentinelRef} className="h-1" />}
          </div>
        )}
      </div>
    </>
  );
}

function ToolChipRow({ tools }: { tools: SetupTool[] }) {
  return (
    <ul className="mt-3 flex flex-wrap gap-2" aria-label="Supported tools">
      {tools.map((tool) => (
        <li
          key={tool.label}
          className={cn(
            "inline-flex items-center gap-1.5 rounded-md bg-slate-50 px-2 py-1 text-xs font-medium text-slate-700",
            "outline outline-slate-950/10",
            "dark:bg-gray-950/40 dark:text-gray-200 dark:outline-gray-700/60",
          )}
        >
          <IntegrationIcon
            integrationName={tool.integrationName}
            iconSlug={tool.iconSlug}
            className="h-3.5 w-3.5"
            size={14}
          />
          {tool.label}
        </li>
      ))}
    </ul>
  );
}

function StarterAppListItem({
  app,
  busy,
  isInstalling,
  onInstall,
  onSelect,
  onCloseInstall,
}: {
  app: AppEntry;
  busy: boolean;
  isInstalling?: boolean;
  onInstall: (app: AppEntry) => void;
  onSelect: (app: AppEntry) => void;
  onCloseInstall: () => void;
}) {
  return (
    <>
      <div onClick={() => onSelect(app)} className={cn("cursor-pointer px-3 py-2.5", homeListCardClassName)}>
        <div className="flex min-h-[30px] items-center justify-between gap-3">
          <div className="flex min-w-0 flex-1 items-center gap-3">
            <div className="shrink-0">
              <LeadIcon icon={app.icon} integrations={app.integrations} size="lg" />
            </div>
            <p className={cn(homeCardTitleClassName, "text-sm")}>{app.title}</p>
          </div>
          <div className="flex shrink-0 items-center gap-2">
            <Button
              type="button"
              variant="outline"
              size="icon-sm"
              onClick={(e) => {
                e.stopPropagation();
                onSelect(app);
              }}
              aria-label={`Preview ${app.title}`}
            >
              <Eye className="h-4 w-4" />
            </Button>
            <Button
              type="button"
              size="sm"
              onClick={(e) => {
                e.stopPropagation();
                onInstall(app);
              }}
              disabled={busy}
            >
              Install
              <ArrowRight />
            </Button>
          </div>
        </div>
      </div>
      {isInstalling && <InstallProgressPanel app={app} onClose={onCloseInstall} />}
    </>
  );
}

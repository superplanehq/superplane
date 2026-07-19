import { Button } from "@/components/ui/button";
import { RequirePermission } from "@/components/PermissionGate";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { cn } from "@/lib/utils";
import { ArrowLeft, ArrowRight, Eye, Factory, Plus } from "lucide-react";
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

/**
 * Storybook-only fresh-org landing POC: Software Factory hero first, with
 * subtle secondary paths for blank apps and the existing starter catalog.
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
          Phase 1 onboarding (connect VCS, LLM, pick template, start Coding Agent) will live here. This screen is a
          Storybook placeholder for now.
        </p>
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

      <div className="mx-auto w-full max-w-3xl px-8 py-16">
        <div
          className={cn(
            "rounded-xl bg-white px-8 py-10 outline outline-slate-950/10",
            "dark:bg-gray-900 dark:outline-gray-700/70",
          )}
        >
          <div className="mb-6 flex h-12 w-12 items-center justify-center rounded-lg bg-slate-100 dark:bg-gray-800">
            <Factory className="h-6 w-6 text-slate-700 dark:text-gray-200" aria-hidden />
          </div>
          <h1 className={homePageTitleClassName}>Set up your Software Factory</h1>
          <p className={cn(homePageSubtitleClassName, "max-w-xl")}>
            Go from labeled issues to plan, PR, and CI babysitting—with a guided factory workflow as your first path in
            SuperPlane.
          </p>
          <div className="mt-8">
            <Button type="button" size="lg" onClick={() => setShowFactoryStub(true)}>
              Get started
              <ArrowRight />
            </Button>
          </div>
        </div>

        <div className="mt-10 flex flex-wrap items-center justify-center gap-x-6 gap-y-2 text-sm">
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
            {showCatalog ? "Hide starter apps" : "Browse starter apps"}
          </button>
        </div>

        {showCatalog && (
          <div className="mt-8 flex flex-col gap-3">
            <p className="text-center text-xs font-medium uppercase tracking-wide text-gray-400 dark:text-gray-500">
              Starter apps
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

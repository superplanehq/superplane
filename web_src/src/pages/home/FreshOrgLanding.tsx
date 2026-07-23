import { Button } from "@/components/ui/button";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { cn } from "@/lib/utils";
import { ArrowRight } from "lucide-react";
import { useEffect, useRef, useState, type RefObject } from "react";

import { LeadIcon, type AppEntry } from "./AppDetailModal";
import { APP_CATALOG } from "./appCatalog";
import { FactorySetupPanel } from "./FactorySetupPanel";
import { homeListCardClassName, homePageSubtitleClassName, homePageTitleClassName } from "./homePageStyles";
import { InstallProgressPanel } from "./InstallProgressPanel";
import type { CanvasFolderData } from "./types";
import { useCreateApp } from "./useCreateApp";

interface FreshOrgLandingProps {
  folder?: CanvasFolderData;
  folderContextPending?: boolean;
  title?: string;
}

/**
 * Factory-first new-app landing prototype (Storybook only via
 * `PrototypeNewAppPage`). Production `/apps/new` still uses `ZeroStatePage`.
 *
 * Setup Factory primary CTA, with quiet escape hatches for blank apps and the
 * starter catalog (catalog hidden until Browse). Factory setup requires
 * integrations, a repository, and a starting task before Run.
 */
export function FreshOrgLanding({
  folder,
  folderContextPending = false,
  title = "Create a new app",
}: FreshOrgLandingProps) {
  const { createApp, isSaving } = useCreateApp({ folder });
  const [showFactorySetup, setShowFactorySetup] = useState(false);
  const [showCatalog, setShowCatalog] = useState(false);
  const [visibleCount, setVisibleCount] = useState(7);
  const [installingApp, setInstallingApp] = useState<AppEntry | null>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const busy = folderContextPending || isSaving;
  const inFocusedSetup = showFactorySetup || installingApp !== null;
  const visible = APP_CATALOG.slice(0, visibleCount);

  useEffect(() => {
    if (!showCatalog || inFocusedSetup) return;
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
  }, [showCatalog, inFocusedSetup]);

  return (
    <div className="mx-auto w-full max-w-3xl px-8 py-14 lg:py-20">
      <h1 className={cn(homePageTitleClassName, "text-2xl text-gray-800")}>{title}</h1>
      <p className={cn(homePageSubtitleClassName, "mt-3 max-w-lg font-normal leading-normal text-gray-600")}>
        Set up a Software Factory to automate coding work with agents, from trigger to pull request. Or start from a
        blank app or starter template instead.
      </p>
      {!inFocusedSetup && (
        <div className="mt-7">
          <Button
            type="button"
            size="lg"
            disabled={busy}
            onClick={() => {
              setShowCatalog(false);
              setInstallingApp(null);
              setShowFactorySetup(true);
            }}
          >
            Setup Factory
            <ArrowRight />
          </Button>
        </div>
      )}

      {showFactorySetup && (
        <FactorySetupPanel
          busy={busy}
          onCancel={() => setShowFactorySetup(false)}
          onInstall={(selections, repository, startingTaskPrompt) => {
            if (busy) return;
            void createApp("Software Factory", {
              factorySetup: { repository, integrations: selections, startingTaskPrompt },
            });
          }}
        />
      )}

      {installingApp && (
        <InstallProgressPanel app={installingApp} folder={folder} onClose={() => setInstallingApp(null)} />
      )}

      {!inFocusedSetup && (
        <BlankOrBrowseLinks
          busy={busy}
          showCatalog={showCatalog}
          onCreateBlank={() => {
            if (busy) return;
            void createApp(generateCanvasName());
          }}
          onToggleCatalog={() => setShowCatalog((open) => !open)}
        />
      )}

      {!inFocusedSetup && showCatalog && (
        <StarterAppsCatalog
          apps={visible}
          busy={busy}
          hasMore={visibleCount < APP_CATALOG.length}
          sentinelRef={sentinelRef}
          onSetup={(app) => {
            if (busy) return;
            setShowFactorySetup(false);
            setInstallingApp(app);
          }}
        />
      )}
    </div>
  );
}

function BlankOrBrowseLinks({
  busy,
  showCatalog,
  onCreateBlank,
  onToggleCatalog,
}: {
  busy: boolean;
  showCatalog: boolean;
  onCreateBlank: () => void;
  onToggleCatalog: () => void;
}) {
  return (
    <p className="mt-8 text-sm font-normal text-gray-600 dark:text-gray-400">
      <Button
        type="button"
        variant="link"
        disabled={busy}
        onClick={onCreateBlank}
        className="h-auto p-0 text-sm font-normal text-gray-800 underline decoration-gray-300 underline-offset-4 dark:text-gray-200 dark:decoration-gray-600"
      >
        Create a blank app
      </Button>
      {" or "}
      <Button
        type="button"
        variant="link"
        onClick={onToggleCatalog}
        className="h-auto p-0 text-sm font-normal text-gray-800 underline decoration-gray-300 underline-offset-4 dark:text-gray-200 dark:decoration-gray-600"
        aria-expanded={showCatalog}
      >
        {showCatalog ? "Hide starter apps" : "Browse starter apps"}
      </Button>
    </p>
  );
}

function StarterAppsCatalog({
  apps,
  busy,
  hasMore,
  sentinelRef,
  onSetup,
}: {
  apps: AppEntry[];
  busy: boolean;
  hasMore: boolean;
  sentinelRef: RefObject<HTMLDivElement | null>;
  onSetup: (app: AppEntry) => void;
}) {
  return (
    <div className="mt-10 flex flex-col gap-3">
      <p className="text-xs font-normal text-gray-600 dark:text-gray-400">
        Automation starters (not Software Factory setup)
      </p>
      {apps.map((app) => (
        <StarterAppListItem key={app.repo} app={app} busy={busy} onSetup={() => onSetup(app)} />
      ))}
      {hasMore && <div ref={sentinelRef} className="h-1" />}
    </div>
  );
}

function StarterAppListItem({ app, busy, onSetup }: { app: AppEntry; busy: boolean; onSetup: () => void }) {
  return (
    <div className={cn("px-3 py-2.5", homeListCardClassName)}>
      <div className="flex min-h-[30px] items-center justify-between gap-3">
        <div className="flex min-w-0 flex-1 items-center gap-3">
          <div className="shrink-0">
            <LeadIcon icon={app.icon} integrations={app.integrations} size="lg" />
          </div>
          <p className="text-base font-medium text-slate-900 dark:text-gray-100">{app.title}</p>
        </div>
        <Button type="button" size="sm" onClick={onSetup} disabled={busy}>
          Setup
          <ArrowRight />
        </Button>
      </div>
    </div>
  );
}

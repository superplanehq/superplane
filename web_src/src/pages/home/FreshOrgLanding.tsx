import { Button } from "@/components/ui/button";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { cn } from "@/lib/utils";
import { ArrowRight, Eye } from "lucide-react";
import { useEffect, useRef, useState } from "react";

import { AppDetailModal, LeadIcon, type AppEntry } from "./AppDetailModal";
import { APP_CATALOG } from "./appCatalog";
import { FactorySetupPanel } from "./FactorySetupPanel";
import {
  homeCardTitleClassName,
  homeListCardClassName,
  homePageSubtitleClassName,
  homePageTitleClassName,
} from "./homePageStyles";
import { InstallProgressPanel } from "./InstallProgressPanel";
import type { CanvasFolderData } from "./types";
import { useCreateApp } from "./useCreateApp";

interface FreshOrgLandingProps {
  folder?: CanvasFolderData;
  folderContextPending?: boolean;
}

/**
 * Factory-first new-app landing: Setup Factory primary CTA, with quiet escape
 * hatches for blank apps and the starter catalog (catalog hidden until Browse).
 */
export function FreshOrgLanding({ folder, folderContextPending = false }: FreshOrgLandingProps) {
  const { createApp, isSaving } = useCreateApp({ folder });
  const [showFactorySetup, setShowFactorySetup] = useState(false);
  const [showCatalog, setShowCatalog] = useState(false);
  const [visibleCount, setVisibleCount] = useState(7);
  const [selectedApp, setSelectedApp] = useState<AppEntry | null>(null);
  const [installingApp, setInstallingApp] = useState<AppEntry | null>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const busy = folderContextPending || isSaving || installingApp !== null;
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

      <div className="mx-auto w-full max-w-3xl px-8 py-14 lg:py-20">
        <h1 className={cn(homePageTitleClassName, "text-2xl text-gray-800")}>Create a new app</h1>
        <p className={cn(homePageSubtitleClassName, "mt-3 max-w-lg font-normal leading-normal text-gray-600")}>
          Set up a Software Factory to automate coding work with agents, from trigger to pull request. Or start from a
          blank app or starter template instead.
        </p>
        {!showFactorySetup && (
          <div className="mt-7">
            <Button
              type="button"
              size="lg"
              disabled={busy}
              onClick={() => {
                setShowCatalog(false);
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
            onInstall={() => {
              if (busy) return;
              void createApp("Software Factory");
            }}
            onPreviewWithoutConnecting={() => {
              if (busy) return;
              void createApp("Software Factory");
            }}
          />
        )}

        {!showFactorySetup && (
          <p className="mt-8 text-sm font-normal text-gray-600 dark:text-gray-400">
            <button
              type="button"
              disabled={busy}
              onClick={() => {
                if (busy) return;
                void createApp(generateCanvasName());
              }}
              className="font-normal text-gray-800 underline decoration-gray-300 underline-offset-4 disabled:opacity-50 dark:text-gray-200 dark:decoration-gray-600"
            >
              Create a blank app
            </button>
            {" or "}
            <button
              type="button"
              onClick={() => setShowCatalog((open) => !open)}
              className="font-normal text-gray-800 underline decoration-gray-300 underline-offset-4 dark:text-gray-200 dark:decoration-gray-600"
              aria-expanded={showCatalog}
            >
              {showCatalog ? "Hide starter apps" : "Browse starter apps"}
            </button>
          </p>
        )}

        {!showFactorySetup && showCatalog && (
          <div className="mt-10 flex flex-col gap-3">
            <p className="text-xs font-normal text-gray-600 dark:text-gray-400">
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
                folder={folder}
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
  folder,
}: {
  app: AppEntry;
  busy: boolean;
  isInstalling?: boolean;
  onInstall: (app: AppEntry) => void;
  onSelect: (app: AppEntry) => void;
  onCloseInstall: () => void;
  folder?: CanvasFolderData;
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
      {isInstalling && <InstallProgressPanel app={app} folder={folder} onClose={onCloseInstall} />}
    </>
  );
}

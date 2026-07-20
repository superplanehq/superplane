import { Button } from "@/components/ui/button";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { ArrowRight, Eye, Plus } from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { AppDetailModal, LeadIcon, type AppEntry } from "./AppDetailModal";
import { APP_CATALOG } from "./appCatalog";
import { cn } from "@/lib/utils";
import {
  createActionCardClassName,
  createActionCardDisabledClassName,
  createActionIconClassName,
  createActionIconDisabledClassName,
} from "@/lib/createActionStyles";
import { useCreateApp } from "./useCreateApp";
import { InstallProgressPanel } from "./InstallProgressPanel";
import {
  homeCardTitleClassName,
  homeDividerLabelClassName,
  homeDividerLineClassName,
  homeListCardClassName,
  homePageSubtitleClassName,
  homePageTitleClassName,
} from "./homePageStyles";
import type { CanvasFolderData } from "./types";

interface ZeroStatePageProps {
  folder?: CanvasFolderData;
  folderContextPending?: boolean;
  title?: string;
}

export function ZeroStatePage({ folder, folderContextPending = false, title = "Create New App" }: ZeroStatePageProps) {
  const { createApp, isSaving } = useCreateApp({ folder });
  const [visibleCount, setVisibleCount] = useState(7);
  const [selectedApp, setSelectedApp] = useState<AppEntry | null>(null);
  const [installingApp, setInstallingApp] = useState<AppEntry | null>(null);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const busy = folderContextPending || isSaving || installingApp !== null;

  const visible = APP_CATALOG.slice(0, visibleCount);

  useEffect(() => {
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
  }, []);

  const handleBlankCreate = () => {
    if (busy) return;
    void createApp(generateCanvasName());
  };

  const handleInstall = (app: AppEntry) => {
    if (busy) return;
    setInstallingApp(app);
    setSelectedApp(null);
  };

  return (
    <>
      {selectedApp && (
        <AppDetailModal
          app={selectedApp}
          busy={busy}
          onBack={() => setSelectedApp(null)}
          onInstall={(e) => {
            e.stopPropagation();
            handleInstall(selectedApp);
          }}
          onClose={() => setSelectedApp(null)}
        />
      )}
      <div className="mx-auto w-full max-w-3xl px-8 py-16">
        <div className="mb-10 text-center">
          <h1 className={homePageTitleClassName}>{title}</h1>
          <p className={homePageSubtitleClassName}>Create a blank app or pick one from the catalog below.</p>
        </div>

        <BlankAppButton busy={busy} onCreate={handleBlankCreate} />

        <div className="relative my-8">
          <div className="absolute inset-0 flex items-center">
            <span className={homeDividerLineClassName} />
          </div>
          <div className="relative flex justify-center text-sm">
            <span className={homeDividerLabelClassName}>or begin with a starter app</span>
          </div>
        </div>

        <div className="flex flex-col gap-4">
          {visible.map((app) => (
            <AppListItem
              key={app.repo}
              app={app}
              busy={busy}
              isInstalling={installingApp?.repo === app.repo}
              onSelect={setSelectedApp}
              onInstall={handleInstall}
              onCloseInstall={() => setInstallingApp(null)}
              folder={folder}
            />
          ))}
          {visibleCount < APP_CATALOG.length && <div ref={sentinelRef} className="h-1" />}
        </div>
      </div>
    </>
  );
}

function BlankAppButton({ busy, onCreate }: { busy: boolean; onCreate: () => void }) {
  return (
    <button
      type="button"
      disabled={busy}
      onClick={onCreate}
      className={cn("min-h-[58px] text-left", busy ? createActionCardDisabledClassName : createActionCardClassName)}
    >
      <span className={busy ? createActionIconDisabledClassName : createActionIconClassName}>
        <Plus className="h-4 w-4" strokeWidth={2} aria-hidden />
      </span>
      <div>
        <p className={homeCardTitleClassName}>Start from scratch</p>
      </div>
    </button>
  );
}

function AppListItem({
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
      <div onClick={() => onSelect(app)} className={cn("cursor-pointer px-4 py-3", homeListCardClassName)}>
        <div className="flex min-h-[34px] items-center justify-between gap-4">
          <div className="flex min-w-0 flex-1 items-center gap-4">
            <div className="shrink-0">
              <LeadIcon icon={app.icon} integrations={app.integrations} size="lg" />
            </div>
            <div className="flex min-w-0 items-center gap-2">
              <p className={homeCardTitleClassName}>{app.title}</p>
            </div>
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

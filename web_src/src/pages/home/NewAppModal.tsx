import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Button } from "@/components/ui/button";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { ArrowRight, Plus } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { AppDetailModal, IntegrationIcons, LeadIcon, type AppEntry } from "./AppDetailModal";
import { filterAppCatalog } from "./appCatalog";
import { useCreateApp } from "./useCreateApp";
import { useInstallTemplate } from "./useInstallTemplate";

interface NewAppModalProps {
  open: boolean;
  onClose: () => void;
}

export function NewAppModal({ open, onClose }: NewAppModalProps) {
  const { createApp, isSaving } = useCreateApp({ onCreated: onClose });
  const { installTemplate, isInstalling } = useInstallTemplate();
  const [search, setSearch] = useState("");
  const [selectedApp, setSelectedApp] = useState<AppEntry | null>(null);
  const [visibleCount, setVisibleCount] = useState(7);
  const sentinelRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (!open) {
      setSearch("");
      setSelectedApp(null);
      setVisibleCount(7);
    }
  }, [open]);

  const filtered = useMemo(() => {
    return filterAppCatalog(search);
  }, [search]);

  const visible = search ? filtered : filtered.slice(0, visibleCount);
  const busy = isSaving || isInstalling;

  useEffect(() => {
    const el = sentinelRef.current;
    if (!el || search) return;
    if (typeof IntersectionObserver === "undefined") {
      setVisibleCount(filtered.length);
      return;
    }

    const observer = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) {
          setVisibleCount((prev) => Math.min(prev + 7, filtered.length));
        }
      },
      { rootMargin: "100px" },
    );
    observer.observe(el);
    return () => observer.disconnect();
  }, [filtered.length, search, visibleCount]);

  const handleBlankCreate = () => {
    if (busy) return;
    void createApp(generateCanvasName());
  };

  const handleInstall = (e: React.MouseEvent, app: AppEntry) => {
    e.stopPropagation();
    if (busy) return;
    void installTemplate(app.repo, {
      instructions: app.agentInstructions,
      initialMessage: app.agentInitialMessage,
    });
  };

  const handleClose = () => {
    setSelectedApp(null);
    setSearch("");
    setVisibleCount(7);
    onClose();
  };

  if (selectedApp) {
    return (
      <AppDetailModal
        app={selectedApp}
        busy={busy}
        onBack={() => setSelectedApp(null)}
        onInstall={(e) => handleInstall(e, selectedApp)}
        onClose={handleClose}
      />
    );
  }

  return (
    <CommandDialog
      open={open}
      onOpenChange={(v) => {
        if (!v) handleClose();
      }}
      title="New App"
      description="Create a blank app or install one from the catalog."
      className="top-[12vh] max-h-[min(760px,80vh)] w-[calc(100vw-2rem)] max-w-3xl sm:max-w-3xl translate-y-0 overflow-hidden rounded-xl border border-slate-200 bg-white p-0 shadow-2xl sm:top-[14vh]"
      showCloseButton={false}
    >
      <CommandInput
        value={search}
        onValueChange={(v) => {
          setSearch(v);
        }}
        placeholder="Search apps..."
        className="h-12"
      />
      <div className="border-b border-slate-200 px-3 py-2">
        <BlankAppCommandItem busy={busy} onCreate={handleBlankCreate} />
      </div>
      <CommandList className="max-h-[360px] scroll-py-2 px-3 py-3">
        <CommandEmpty>No apps found.</CommandEmpty>

        {visible.length > 0 && (
          <>
            <CommandGroup heading="Apps">
              {visible.map((app) => (
                <AppCommandItem
                  key={app.repo}
                  app={app}
                  busy={busy}
                  onSelect={setSelectedApp}
                  onInstall={handleInstall}
                />
              ))}
            </CommandGroup>
            {!search && visibleCount < filtered.length && <div ref={sentinelRef} className="h-1" />}
          </>
        )}
      </CommandList>
    </CommandDialog>
  );
}

function BlankAppCommandItem({ busy, onCreate }: { busy: boolean; onCreate: () => void }) {
  return (
    <CommandItem onSelect={onCreate} disabled={busy} className="gap-3 px-3 py-3">
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-slate-100">
        <Plus className="h-4 w-4 text-slate-600" />
      </div>
      <div>
        <p className="text-sm font-medium">Start from scratch</p>
        <p className="text-xs text-muted-foreground">Create a blank app</p>
      </div>
    </CommandItem>
  );
}

function AppCommandItem({
  app,
  busy,
  onInstall,
  onSelect,
}: {
  app: AppEntry;
  busy: boolean;
  onInstall: (e: React.MouseEvent, app: AppEntry) => void;
  onSelect: (app: AppEntry) => void;
}) {
  return (
    <CommandItem
      value={`${app.title} ${app.description} ${app.integrations.join(" ")} ${app.tags.join(" ")}`}
      onSelect={() => onSelect(app)}
      disabled={busy}
      className="gap-3 px-3 py-3"
    >
      <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-slate-100">
        <LeadIcon icon={app.icon} integrations={app.integrations} />
      </div>
      <div className="min-w-0 flex-1">
        <div className="flex items-center gap-2">
          <p className="text-sm font-medium">{app.title}</p>
          <IntegrationIcons integrations={app.integrations} />
        </div>
        <p className="text-xs text-muted-foreground line-clamp-1">{app.description}</p>
      </div>
      <Button
        size="sm"
        className="shrink-0 text-xs"
        onPointerDown={(e) => e.stopPropagation()}
        onClick={(e) => onInstall(e, app)}
        disabled={busy}
      >
        Install
        <ArrowRight className="ml-1 h-3 w-3" />
      </Button>
    </CommandItem>
  );
}

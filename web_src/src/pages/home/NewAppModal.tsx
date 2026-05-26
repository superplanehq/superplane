import {
  CommandDialog,
  CommandEmpty,
  CommandGroup,
  CommandInput,
  CommandItem,
  CommandList,
} from "@/components/ui/command";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { getIntegrationIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { ArrowLeft, ArrowRight, ExternalLink, Plus } from "lucide-react";
import { useCallback, useMemo, useRef, useState } from "react";
import templateManifest from "../../../../templates/manifest.json";
import { useCreateApp } from "./useCreateApp";
import { useInstallTemplate } from "./useInstallTemplate";

interface AppEntry {
  repo: string;
  title: string;
  description: string;
  integrations: string[];
  tags: string[];
  requirements: string[];
}

const allApps: AppEntry[] = templateManifest;

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
  const listRef = useRef<HTMLDivElement>(null);

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return allApps;
    return allApps.filter(
      (t) =>
        t.title.toLowerCase().includes(q) ||
        t.description.toLowerCase().includes(q) ||
        t.integrations.some((i) => i.toLowerCase().includes(q)) ||
        t.tags.some((tag) => tag.toLowerCase().includes(q)),
    );
  }, [search]);

  const visible = search ? filtered : filtered.slice(0, visibleCount);
  const busy = isSaving || isInstalling;

  const handleScroll = useCallback(() => {
    const el = listRef.current;
    if (!el || search) return;
    if (el.scrollTop + el.clientHeight >= el.scrollHeight - 40) {
      setVisibleCount((prev) => Math.min(prev + 7, filtered.length));
    }
  }, [filtered.length, search]);

  const handleBlankCreate = () => {
    if (busy) return;
    void createApp(generateCanvasName());
  };

  const handleInstall = (e: React.MouseEvent, repo: string) => {
    e.stopPropagation();
    if (busy) return;
    void installTemplate(repo);
  };

  const handleClose = () => {
    setSelectedApp(null);
    setSearch("");
    onClose();
  };

  if (selectedApp) {
    return (
      <AppDetailView
        app={selectedApp}
        busy={busy}
        onBack={() => setSelectedApp(null)}
        onInstall={(e) => handleInstall(e, selectedApp.repo)}
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
        <CommandItem onSelect={handleBlankCreate} disabled={busy} className="gap-3 px-3 py-3">
          <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-slate-100">
            <Plus className="h-4 w-4 text-slate-600" />
          </div>
          <div>
            <p className="text-sm font-medium">Start from scratch</p>
            <p className="text-xs text-muted-foreground">Create a blank app</p>
          </div>
        </CommandItem>
      </div>
      <CommandList
        ref={listRef}
        onScroll={handleScroll}
        className="max-h-[360px] scroll-py-2 px-3 py-3"
      >
        <CommandEmpty>No apps found.</CommandEmpty>

        {visible.length > 0 && (
          <>
            <CommandGroup heading="Apps">
              {visible.map((app) => (
                <CommandItem
                  key={app.repo}
                  value={`${app.title} ${app.description} ${app.integrations.join(" ")}`}
                  onSelect={() => setSelectedApp(app)}
                  disabled={busy}
                  className="gap-3 px-3 py-3"
                >
                  <div className="flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-slate-100">
                    <LeadIntegrationIcon integrations={app.integrations} />
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
                    onClick={(e) => handleInstall(e, app.repo)}
                    disabled={busy}
                  >
                    Install
                    <ArrowRight className="ml-1 h-3 w-3" />
                  </Button>
                </CommandItem>
              ))}
            </CommandGroup>
          </>
        )}
      </CommandList>
    </CommandDialog>
  );
}

function AppDetailView({
  app,
  busy,
  onBack,
  onInstall,
  onClose,
}: {
  app: AppEntry;
  busy: boolean;
  onBack: () => void;
  onInstall: (e: React.MouseEvent) => void;
  onClose: () => void;
}) {
  const repoUrl = `https://${app.repo}`;

  return (
    <div className="fixed inset-0 z-[200] flex items-start justify-center pt-[12vh] sm:pt-[14vh]">
      <div className="fixed inset-0 bg-gray-950/20" onClick={onClose} />
      <div className="relative w-[calc(100vw-2rem)] max-w-3xl rounded-xl border border-slate-200 bg-white shadow-2xl">
        <div className="flex items-center gap-2 border-b border-slate-200 px-5 py-3">
          <button
            type="button"
            onClick={onBack}
            className="flex items-center gap-1 text-xs font-medium text-slate-500 hover:text-slate-700"
          >
            <ArrowLeft className="h-3.5 w-3.5" />
            Back
          </button>
        </div>

        <div className="px-6 py-5">
          <div className="flex items-start gap-4">
            <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-xl bg-slate-100">
              <LeadIntegrationIcon integrations={app.integrations} size="lg" />
            </div>
            <div className="min-w-0 flex-1">
              <h3 className="text-lg font-semibold text-slate-900">{app.title}</h3>
              <div className="mt-1.5 flex flex-wrap items-center gap-2">
                <IntegrationIcons integrations={app.integrations} />
                {app.tags.map((tag) => (
                  <span
                    key={tag}
                    className="rounded-full bg-slate-100 px-2.5 py-0.5 text-xs font-medium text-slate-600"
                  >
                    {tag}
                  </span>
                ))}
              </div>
            </div>
          </div>

          <div className="mt-5">
            <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-400">Description</h4>
            <p className="mt-1.5 text-sm leading-relaxed text-slate-600">{app.description}</p>
          </div>

          {app.requirements.length > 0 && (
            <div className="mt-4">
              <h4 className="text-xs font-semibold uppercase tracking-wide text-slate-400">Requirements</h4>
              <ul className="mt-1.5 space-y-1">
                {app.requirements.map((req) => (
                  <li key={req} className="flex items-start gap-2 text-sm text-slate-600">
                    <span className="mt-1.5 h-1 w-1 shrink-0 rounded-full bg-slate-400" />
                    {req}
                  </li>
                ))}
              </ul>
            </div>
          )}
        </div>

        <div className="flex items-center justify-between border-t border-slate-200 px-6 py-4">
          <a
            href={repoUrl}
            target="_blank"
            rel="noopener noreferrer"
            className="flex items-center gap-1 text-xs font-medium text-slate-500 hover:text-slate-700"
          >
            <ExternalLink className="h-3.5 w-3.5" />
            View on GitHub
          </a>
          <Button onClick={onInstall} disabled={busy}>
            Install
            <ArrowRight className="ml-1.5 h-4 w-4" />
          </Button>
        </div>
      </div>
    </div>
  );
}

function LeadIntegrationIcon({ integrations, size = "sm" }: { integrations: string[]; size?: "sm" | "lg" }) {
  const first = integrations[0];
  const cls = size === "lg" ? "h-7 w-7" : "h-5 w-5";
  if (!first) return <Plus className={`${cls} text-slate-400`} />;
  const icon = getIntegrationIconSrc(first.toLowerCase());
  if (!icon) return <Plus className={`${cls} text-slate-400`} />;
  return <img src={icon} alt={first} className={cls} />;
}

function IntegrationIcons({ integrations }: { integrations: string[] }) {
  if (integrations.length === 0) return null;

  return (
    <div className="flex items-center gap-1 shrink-0">
      {integrations.map((name) => {
        const iconSrc = getIntegrationIconSrc(name.toLowerCase());
        if (!iconSrc) return null;
        return (
          <Tooltip key={name}>
            <TooltipTrigger asChild>
              <span className="inline-block h-3.5 w-3.5 shrink-0">
                <img src={iconSrc} alt={name} className="h-full w-full object-contain" />
              </span>
            </TooltipTrigger>
            <TooltipContent side="bottom">
              <span className="capitalize">{name}</span>
            </TooltipContent>
          </Tooltip>
        );
      })}
    </div>
  );
}

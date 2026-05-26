import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { generateCanvasName } from "@/lib/canvasNameGenerator";
import { getIntegrationIconSrc } from "@/ui/componentSidebar/integrationIconMaps";
import { ArrowRight, Plus, Search } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
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
  agentInstructions: string;
}

const allApps: AppEntry[] = templateManifest;

interface ZeroStatePageProps {
  userName: string;
}

export function ZeroStatePage({ userName }: ZeroStatePageProps) {
  const { createApp, isSaving } = useCreateApp();
  const { installTemplate, isInstalling } = useInstallTemplate();
  const [search, setSearch] = useState("");
  const [visibleCount, setVisibleCount] = useState(7);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const busy = isSaving || isInstalling;

  const firstName = userName.split(" ")[0] || "there";

  const filtered = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return allApps;
    return allApps.filter(
      (t) =>
        t.title.toLowerCase().includes(q) ||
        t.description.toLowerCase().includes(q) ||
        t.integrations.some((i) => i.toLowerCase().includes(q)),
    );
  }, [search]);

  const visible = search ? filtered : filtered.slice(0, visibleCount);

  useEffect(() => {
    const el = sentinelRef.current;
    if (!el || search) return;
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
  }, [filtered.length, search]);

  const handleBlankCreate = () => {
    if (busy) return;
    void createApp(generateCanvasName());
  };

  const handleInstall = (app: AppEntry) => {
    if (busy) return;
    void installTemplate(app.repo, app.agentInstructions);
  };

  return (
    <div className="mx-auto w-full max-w-3xl px-8 py-16">
      <div className="mb-10 text-center">
        <h1 className="text-2xl font-semibold text-slate-900">Hi {firstName}, let's get you started</h1>
        <p className="mt-2 text-sm text-slate-500">Create a blank app or pick one from the catalog below.</p>
      </div>

      <button
        type="button"
        disabled={busy}
        onClick={handleBlankCreate}
        className="flex w-full items-center gap-4 rounded-xl border border-slate-200 bg-white p-5 text-left transition-colors hover:bg-slate-50 disabled:opacity-50"
      >
        <div className="flex h-11 w-11 shrink-0 items-center justify-center rounded-xl bg-slate-100">
          <Plus className="h-5 w-5 text-slate-600" />
        </div>
        <div>
          <p className="text-base font-medium text-slate-900">Start from scratch</p>
          <p className="mt-0.5 text-sm text-slate-500">Create a blank app and build your workflow</p>
        </div>
      </button>

      <div className="relative my-8">
        <div className="absolute inset-0 flex items-center">
          <span className="w-full border-t border-slate-200" />
        </div>
        <div className="relative flex justify-center text-sm">
          <span className="bg-slate-100 px-3 text-slate-500">Or install an app</span>
        </div>
      </div>

      <div className="mb-4">
        <div className="relative">
          <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-400" />
          <input
            type="text"
            placeholder="Search apps..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-slate-200 bg-white py-2.5 pl-9 pr-3 text-sm text-slate-900 placeholder:text-slate-400 focus:border-slate-300 focus:outline-none focus:ring-1 focus:ring-slate-300"
          />
        </div>
      </div>

      <div className="flex flex-col gap-2">
        {visible.map((app) => (
          <div
            key={app.repo}
            className="flex items-center gap-4 rounded-xl border border-slate-200 bg-white p-4 transition-colors hover:bg-slate-50"
          >
            <div className="flex h-9 w-9 shrink-0 items-center justify-center rounded-lg bg-slate-100">
              <LeadIcon integrations={app.integrations} />
            </div>
            <div className="min-w-0 flex-1">
              <div className="flex items-center gap-2">
                <p className="text-sm font-medium text-slate-900">{app.title}</p>
                <IntegrationIcons integrations={app.integrations} />
              </div>
              <p className="mt-0.5 text-xs text-slate-500 line-clamp-1">{app.description}</p>
            </div>
            <Button size="sm" className="shrink-0 text-xs" onClick={() => handleInstall(app)} disabled={busy}>
              Install
              <ArrowRight className="ml-1 h-3 w-3" />
            </Button>
          </div>
        ))}
        {!search && visibleCount < filtered.length && <div ref={sentinelRef} className="h-1" />}
        {search && filtered.length === 0 && (
          <p className="py-8 text-center text-sm text-slate-500">No apps matching &ldquo;{search}&rdquo;</p>
        )}
      </div>
    </div>
  );
}

function LeadIcon({ integrations }: { integrations: string[] }) {
  const first = integrations[0];
  if (!first) return <Plus className="h-4 w-4 text-slate-400" />;
  const icon = getIntegrationIconSrc(first.toLowerCase());
  if (!icon) return <Plus className="h-4 w-4 text-slate-400" />;
  return <img src={icon} alt={first} className="h-5 w-5" />;
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

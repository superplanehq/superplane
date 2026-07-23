import type { OrganizationsIntegration } from "@/api-client";
import { Button } from "@/components/ui/button";
import { getIntegrationTypeDisplayName } from "@/lib/integrationDisplayName";
import { cn } from "@/lib/utils";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { ArrowLeftRight, Check } from "lucide-react";
import { useState } from "react";

import type { HomeIntegrationStatus, HomeIntegrationStatusKind } from "./homeIntegrationStatus";

const STATUS_DOT_COLORS: Record<string, string> = {
  ready: "bg-emerald-500",
  pending: "bg-amber-500",
  error: "bg-red-500",
};

function StatusDot({ state }: { state?: string }) {
  const color = state ? STATUS_DOT_COLORS[state] : undefined;
  if (!color) return null;
  return <span className={`inline-block h-1.5 w-1.5 shrink-0 rounded-full ${color}`} />;
}

const STATUS_LABEL_CLASS: Record<HomeIntegrationStatusKind, string> = {
  ready: "text-emerald-700 dark:text-emerald-300",
  pending: "text-amber-700 dark:text-amber-300",
  error: "text-red-700 dark:text-red-300",
  none: "text-gray-400 dark:text-gray-500",
};

function InstanceSwitchPopover({
  open,
  onOpenChange,
  instances,
  selectedId,
  onSelect,
  onConfigure,
  onCreateNew,
}: {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  instances: OrganizationsIntegration[];
  selectedId?: string;
  onSelect: (id: string, name: string) => void;
  onConfigure: (id: string) => void;
  onCreateNew: () => void;
}) {
  return (
    <Popover open={open} onOpenChange={onOpenChange}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="xs"
          className="h-7 w-7 shrink-0 p-0 text-slate-500 hover:text-slate-800 dark:text-gray-400 dark:hover:text-gray-100"
          aria-label="Switch integration instance"
        >
          <ArrowLeftRight className="h-3.5 w-3.5" />
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-64 p-1">
        <div className="px-2 py-1.5 text-[11px] font-medium uppercase tracking-wide text-slate-400 dark:text-gray-500">
          Choose instance
        </div>
        <div className="max-h-56 overflow-y-auto">
          {instances.map((instance) => {
            const id = instance.metadata?.id;
            const instanceName = instance.metadata?.name || id || "Unnamed";
            if (!id) return null;
            const selected = id === selectedId;
            const ready = instance.status?.state === "ready";
            return (
              <button
                key={id}
                type="button"
                className={cn(
                  "flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs",
                  "hover:bg-slate-100 dark:hover:bg-gray-800",
                  selected && "bg-slate-50 dark:bg-gray-800/80",
                )}
                onClick={() => {
                  if (!ready) {
                    onConfigure(id);
                    return;
                  }
                  onSelect(id, instanceName);
                }}
              >
                <StatusDot state={instance.status?.state} />
                <span className="min-w-0 flex-1 truncate text-slate-800 dark:text-gray-100">{instanceName}</span>
                {selected ? <Check className="h-3.5 w-3.5 shrink-0 text-emerald-600 dark:text-emerald-400" /> : null}
              </button>
            );
          })}
        </div>
        <div className="mt-1 border-t border-slate-200 pt-1 dark:border-gray-700">
          <button
            type="button"
            className="w-full rounded-md px-2 py-1.5 text-left text-xs text-slate-600 hover:bg-slate-100 dark:text-gray-300 dark:hover:bg-gray-800"
            onClick={onCreateNew}
          >
            Create new…
          </button>
        </div>
      </PopoverContent>
    </Popover>
  );
}

export function HomeIntegrationConnectRow({
  name,
  status,
  instances,
  selectedId,
  selectedName,
  onConnect,
  onConfigure,
  onSelect,
  onCreateNew,
}: {
  name: string;
  status: HomeIntegrationStatus;
  instances: OrganizationsIntegration[];
  selectedId?: string;
  selectedName?: string;
  onConnect: () => void;
  onConfigure: (id: string) => void;
  onSelect: (id: string, name: string) => void;
  onCreateNew: () => void;
}) {
  const [switchOpen, setSwitchOpen] = useState(false);
  const displayName = getIntegrationTypeDisplayName(undefined, name) || name.charAt(0).toUpperCase() + name.slice(1);
  const canSwitch = instances.length > 1;

  return (
    <div className="flex min-h-7 items-center gap-2 px-3 py-2.5">
      <IntegrationIcon integrationName={name} className="h-4 w-4 shrink-0" size={16} />
      <span className="shrink-0 truncate text-sm font-medium text-slate-900 dark:text-gray-100">{displayName}</span>
      <div className="flex min-w-0 flex-1 items-center gap-1.5">
        <span className={cn("shrink-0 text-xs font-medium", STATUS_LABEL_CLASS[status.kind])}>{status.label}</span>
        {status.kind === "ready" && selectedName && (
          <span className="truncate text-xs text-slate-500 dark:text-gray-400">{selectedName}</span>
        )}
      </div>
      {status.kind === "none" && (
        <Button type="button" variant="outline" size="xs" className="shrink-0" onClick={onConnect}>
          Connect
        </Button>
      )}
      {status.kind === "pending" && (
        <Button type="button" variant="outline" size="xs" className="shrink-0" onClick={onConnect}>
          Configure
        </Button>
      )}
      {status.kind === "error" && status.configureId && (
        <Button
          type="button"
          variant="outline"
          size="xs"
          className="shrink-0"
          onClick={() => onConfigure(status.configureId!)}
        >
          Configure
        </Button>
      )}
      {canSwitch && (
        <InstanceSwitchPopover
          open={switchOpen}
          onOpenChange={setSwitchOpen}
          instances={instances}
          selectedId={selectedId}
          onSelect={(id, instanceName) => {
            onSelect(id, instanceName);
            setSwitchOpen(false);
          }}
          onConfigure={(id) => {
            setSwitchOpen(false);
            onConfigure(id);
          }}
          onCreateNew={() => {
            setSwitchOpen(false);
            onCreateNew();
          }}
        />
      )}
    </div>
  );
}

export { StatusDot };

import { useMemo, useState } from "react";
import {
  AlarmClock,
  Bug,
  ChevronDown,
  ChevronRight,
  Clock,
  Code2,
  Cuboid,
  Filter,
  GitBranch,
  GitMerge,
  Globe,
  Hand,
  HardDrive,
  Loader,
  Play,
  Plug,
  RefreshCw,
  Search,
  Server,
  Settings2,
  Split,
  Terminal,
  TriangleAlert,
  Webhook,
  Zap,
  X,
  type LucideIcon,
} from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";

type ComponentKind = "trigger" | "action";

type CatalogItem = {
  id: string;
  name: string;
  kind: ComponentKind;
  icon: LucideIcon;
};

type CatalogCategory = {
  id: string;
  name: string;
  icon: LucideIcon;
  connected?: boolean;
  items: CatalogItem[];
};

const CATALOG: CatalogCategory[] = [
  {
    id: "core",
    name: "Core",
    icon: Zap,
    connected: true,
    items: [
      { id: "manual-run", name: "Manual Run", kind: "trigger", icon: Play },
      { id: "on-error", name: "On Error", kind: "trigger", icon: TriangleAlert },
      { id: "schedule", name: "Schedule", kind: "trigger", icon: Clock },
      { id: "webhook", name: "Webhook", kind: "trigger", icon: Webhook },
      { id: "approval", name: "Approval", kind: "action", icon: Hand },
      { id: "filter", name: "Filter", kind: "action", icon: Filter },
      { id: "for-each", name: "For Each", kind: "action", icon: RefreshCw },
      { id: "graphql", name: "GraphQL Request", kind: "action", icon: Cuboid },
      { id: "http", name: "HTTP Request", kind: "action", icon: Globe },
      { id: "if", name: "If", kind: "action", icon: Split },
      { id: "loop", name: "Loop", kind: "action", icon: Loader },
      { id: "merge", name: "Merge", kind: "action", icon: GitMerge },
      { id: "ssh", name: "SSH Command", kind: "action", icon: Terminal },
      { id: "time-gate", name: "Time Gate", kind: "action", icon: Clock },
      { id: "wait", name: "Wait", kind: "action", icon: AlarmClock },
    ],
  },
  {
    id: "runners",
    name: "Runners",
    icon: Server,
    connected: true,
    items: [
      { id: "run-bash", name: "Run Bash", kind: "action", icon: Code2 },
      { id: "run-claude", name: "Run Claude Code", kind: "action", icon: Code2 },
      { id: "run-js", name: "Run JavaScript", kind: "action", icon: Code2 },
      { id: "run-python", name: "Run Python", kind: "action", icon: Code2 },
      { id: "run-shell", name: "Run Shell Commands", kind: "action", icon: Terminal },
    ],
  },
  {
    id: "debugging",
    name: "Debugging",
    icon: Bug,
    connected: true,
    items: [
      { id: "log", name: "Log", kind: "action", icon: Terminal },
      { id: "breakpoint", name: "Breakpoint", kind: "action", icon: GitBranch },
    ],
  },
  {
    id: "memory",
    name: "Memory",
    icon: HardDrive,
    connected: true,
    items: [
      { id: "read-memory", name: "Read Memory", kind: "action", icon: HardDrive },
      { id: "write-memory", name: "Write Memory", kind: "action", icon: HardDrive },
    ],
  },
];

function KindLabel({ kind }: { kind: ComponentKind }) {
  return (
    <span className={cn("text-[11px] font-medium", kind === "trigger" ? "text-[#2563eb]" : "text-[#16a34a]")}>
      {kind === "trigger" ? "Trigger" : "Action"}
    </span>
  );
}

export function SelectComponentSidebar({ onClose }: { onClose: () => void }) {
  const [query, setQuery] = useState("");
  const [expanded, setExpanded] = useState<Record<string, boolean>>({
    core: true,
    runners: true,
    debugging: false,
    memory: false,
  });

  const filtered = useMemo(() => {
    const normalized = query.trim().toLowerCase();
    if (!normalized) return CATALOG;
    return CATALOG.map((category) => ({
      ...category,
      items: category.items.filter((item) => item.name.toLowerCase().includes(normalized)),
    })).filter((category) => category.items.length > 0);
  }, [query]);

  function toggleCategory(id: string) {
    setExpanded((current) => ({ ...current, [id]: !current[id] }));
  }

  return (
    <aside className="flex h-full w-[320px] shrink-0 flex-col border-l border-border bg-background">
      <div className="flex items-center justify-between gap-3 border-b border-border px-4 py-3">
        <h2 className="text-[14px] font-semibold tracking-[-0.01em] text-foreground">Select Component</h2>
        <button
          type="button"
          onClick={onClose}
          className="flex size-7 shrink-0 cursor-pointer items-center justify-center rounded-md text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          aria-label="Close component picker"
        >
          <X className="size-4" strokeWidth={1.75} />
        </button>
      </div>

      <div className="flex items-center gap-2 border-b border-border px-3 py-2.5">
        <div className="relative min-w-0 flex-1">
          <Search
            className="pointer-events-none absolute top-1/2 left-2.5 size-3.5 -translate-y-1/2 text-muted-foreground"
            strokeWidth={1.75}
          />
          <Input
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder="Filter components..."
            className="h-8 bg-background pl-8 text-[13px] shadow-none"
          />
        </div>
        <button
          type="button"
          className="flex size-8 shrink-0 items-center justify-center rounded-md border border-border text-muted-foreground transition-colors hover:bg-accent hover:text-foreground"
          aria-label="Filter options"
        >
          <Settings2 className="size-3.5" strokeWidth={1.75} />
        </button>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto px-2 py-2">
        {filtered.map((category) => {
          const CategoryIcon = category.icon;
          const isOpen = Boolean(expanded[category.id]) || query.trim().length > 0;
          const Chevron = isOpen ? ChevronDown : ChevronRight;

          return (
            <div key={category.id} className="mb-1">
              <button
                type="button"
                onClick={() => toggleCategory(category.id)}
                className="flex w-full cursor-pointer items-center gap-2 rounded-md px-2 py-2 text-left transition-colors hover:bg-accent/60"
              >
                <Chevron className="size-3.5 shrink-0 text-muted-foreground" strokeWidth={1.75} />
                <CategoryIcon className="size-3.5 shrink-0 text-muted-foreground" strokeWidth={1.75} />
                <span className="min-w-0 flex-1 truncate text-[13px] font-medium text-foreground">{category.name}</span>
                <span className="mx-2 h-px min-w-4 flex-1 bg-border" aria-hidden />
                <Plug
                  className={cn("size-3.5 shrink-0", category.connected ? "text-[#16a34a]" : "text-muted-foreground")}
                  strokeWidth={1.75}
                />
              </button>

              {isOpen ? (
                <ul className="pb-1">
                  {category.items.map((item) => {
                    const ItemIcon = item.icon;
                    return (
                      <li key={item.id}>
                        <button
                          type="button"
                          className="flex w-full cursor-pointer items-center gap-2.5 rounded-md px-2 py-1.5 text-left transition-colors hover:bg-accent/70"
                        >
                          <ItemIcon className="size-3.5 shrink-0 text-muted-foreground" strokeWidth={1.75} />
                          <span className="min-w-0 flex-1 truncate text-[13px] text-foreground">{item.name}</span>
                          <KindLabel kind={item.kind} />
                        </button>
                      </li>
                    );
                  })}
                </ul>
              ) : null}
            </div>
          );
        })}

        {filtered.length === 0 ? (
          <p className="px-3 py-8 text-center text-[13px] text-muted-foreground">No components match.</p>
        ) : null}
      </div>
    </aside>
  );
}

export function NewComponentButton({ onClick }: { onClick: () => void }) {
  return (
    <Button
      type="button"
      size="sm"
      variant="outline"
      className="absolute top-3 right-3 z-10 shadow-sm"
      onClick={onClick}
    >
      New Component
    </Button>
  );
}

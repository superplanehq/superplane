import type { ReactNode } from "react";
import type { LucideIcon } from "lucide-react";
import {
  ArrowLeft,
  BarChart3,
  Bell,
  Blocks,
  CircleUser,
  Cpu,
  Grid3x3,
  Key,
  KeyRound,
  Plug,
  Settings,
  Users,
  Workflow,
} from "lucide-react";

import { cn } from "@/lib/utils";

import type { AccountRedesignPageId } from "./accountProfileRedesignMocks";

interface RedesignNavItem {
  id: string;
  label: string;
  Icon: LucideIcon;
  page?: AccountRedesignPageId;
  keywords?: string[];
}

interface RedesignNavGroup {
  id: string;
  label: string;
  items: RedesignNavItem[];
}

const REDESIGN_NAV_GROUPS: RedesignNavGroup[] = [
  {
    id: "account",
    label: "Account",
    items: [
      {
        id: "account-profile",
        label: "Account",
        Icon: CircleUser,
        page: "profile",
        keywords: ["account", "preferences", "profile", "security", "access", "theme"],
      },
      { id: "account-notifications", label: "Notifications", Icon: Bell },
    ],
  },
  {
    id: "workspace",
    label: "Workspace",
    items: [
      { id: "workspace-general", label: "General", Icon: Grid3x3 },
      { id: "workspace-repository", label: "Repository", Icon: Blocks },
      { id: "workspace-automations", label: "Automations", Icon: Workflow },
      { id: "workspace-models", label: "Models", Icon: Cpu },
      { id: "workspace-spending", label: "Spending", Icon: BarChart3 },
    ],
  },
  {
    id: "organization",
    label: "Organization",
    items: [
      { id: "organization-general", label: "General", Icon: Settings },
      { id: "organization-members", label: "Members", Icon: Users },
      { id: "organization-integrations", label: "Integrations", Icon: Plug },
      { id: "organization-api-keys", label: "API keys", Icon: KeyRound },
      { id: "organization-secrets", label: "Secrets", Icon: Key },
      { id: "organization-spending", label: "Spending", Icon: BarChart3 },
    ],
  },
];

export function AccountProfileRedesignShell({
  activePage,
  navQuery,
  onNavQueryChange,
  onSelectPage,
  children,
}: {
  activePage: AccountRedesignPageId;
  navQuery: string;
  onNavQueryChange: (query: string) => void;
  onSelectPage: (page: AccountRedesignPageId) => void;
  children: ReactNode;
}) {
  const query = navQuery.trim().toLowerCase();

  return (
    <div
      className="flex h-svh min-h-0 w-full overflow-hidden bg-background text-foreground"
      data-testid="account-redesign-layout"
    >
      <aside
        className="flex h-full w-[240px] shrink-0 flex-col overflow-y-auto border-r border-sidebar-border bg-sidebar text-sidebar-foreground"
        data-testid="account-redesign-sidebar"
      >
        <div className="border-b border-sidebar-border px-3 py-3">
          <span className="inline-flex h-8 items-center gap-2 rounded-md px-2.5 text-[13px] tracking-[-0.01em] text-muted-foreground">
            <ArrowLeft className="size-3.5" aria-hidden />
            Back to workspace
          </span>
        </div>
        <div className="px-3 pt-3">
          <label className="sr-only" htmlFor="account-redesign-find">
            Find settings
          </label>
          <input
            id="account-redesign-find"
            data-testid="account-redesign-find"
            value={navQuery}
            onChange={(event) => onNavQueryChange(event.target.value)}
            placeholder="Find settings"
            className="h-8 w-full rounded-md border border-border bg-background px-2.5 text-[13px] text-foreground outline-none placeholder:text-muted-foreground focus-visible:border-foreground/30"
          />
        </div>
        <nav className="flex min-h-0 flex-1 flex-col gap-5 overflow-y-auto px-2 py-4">
          {REDESIGN_NAV_GROUPS.map((group) => {
            const items = group.items.filter((item) => {
              if (!query) {
                return true;
              }
              const haystack = [item.label, ...(item.keywords ?? [])].join(" ").toLowerCase();
              return haystack.includes(query);
            });
            if (items.length === 0) {
              return null;
            }
            return (
              <section key={group.id}>
                <h2 className="px-2.5 pb-1 text-[11px] font-medium uppercase tracking-[0.08em] text-muted-foreground">
                  {group.label}
                </h2>
                <ul className="flex flex-col gap-0.5">
                  {items.map((item) => {
                    const Icon = item.Icon;
                    const isActive = item.page === activePage || (item.page === "profile" && activePage === "security");
                    return (
                      <li key={item.id}>
                        <button
                          type="button"
                          disabled={!item.page}
                          onClick={() => {
                            if (item.page) {
                              onSelectPage(item.page);
                            }
                          }}
                          className={cn(
                            "group flex w-full items-center gap-2 rounded-md px-2.5 py-1.5 text-left text-[13px] tracking-[-0.01em] text-foreground/80 hover:bg-sidebar-accent hover:text-foreground",
                            isActive && "bg-sidebar-accent font-medium text-foreground",
                            !item.page && "cursor-default opacity-50 hover:bg-transparent hover:text-foreground/80",
                          )}
                          data-testid={`account-redesign-nav-${item.id}`}
                        >
                          <Icon className="size-[15px] shrink-0 opacity-80" strokeWidth={1.75} aria-hidden />
                          <span>{item.label}</span>
                        </button>
                      </li>
                    );
                  })}
                </ul>
              </section>
            );
          })}
        </nav>
      </aside>
      <main className="min-h-0 min-w-0 flex-1 overflow-y-auto bg-sidebar dark:bg-background">{children}</main>
    </div>
  );
}

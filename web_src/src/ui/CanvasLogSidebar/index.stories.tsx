import type { Meta, StoryObj } from "@storybook/react";
import { useEffect, useMemo, useRef, useState } from "react";

import {
  CanvasLogSidebar,
  type LogCounts,
  type LogEntry,
  type LogScopeFilter,
  type LogTypeFilter,
  type LogRunItem,
} from "./index";
import { Button } from "@/components/ui/button";

const meta = {
  title: "UI/CanvasLogSidebar",
  component: CanvasLogSidebar,
  parameters: {
    layout: "fullscreen",
  },
  tags: ["autodocs"],
} satisfies Meta<typeof CanvasLogSidebar>;

export default meta;
type Story = StoryObj<typeof meta>;

const sampleEntries: LogEntry[] = [
  {
    id: "log-1",
    type: "success",
    title: "Workflow deployed successfully",
    timestamp: "Tue 2025-12-30 10:15:52",
    source: "canvas",
    searchText: "workflow deployed",
  },
  {
    id: "log-2",
    type: "warning",
    title: "Rate limit nearing threshold",
    timestamp: "Tue 2025-12-30 10:16:24",
    source: "runs",
    searchText: "rate limit warning",
  },
  {
    id: "log-3",
    type: "error",
    title: "Webhook signature mismatch",
    timestamp: "Tue 2025-12-30 10:17:03",
    source: "runs",
    searchText: "webhook signature mismatch",
  },
  {
    id: "log-4",
    type: "run",
    title: "Run #1934: Checkout flow",
    timestamp: "Tue 2025-12-30 10:18:19",
    source: "runs",
    runItems: [
      {
        id: "run-1",
        type: "success",
        title: (
          <span>
            Payment captured for{" "}
            <a className="text-blue-600 underline" href="#">
              order #1042
            </a>
          </span>
        ),
        timestamp: "Tue 2025-12-30 10:18:42",
        detail: "Stripe charge confirmed and stored.",
        searchText: "payment captured order 1042",
      },
      {
        id: "run-2",
        type: "warning",
        title: "Inventory low for SKU-AX9",
        timestamp: "Tue 2025-12-30 10:19:10",
        detail: "Only 3 units left in stock.",
        searchText: "inventory low sku ax9",
      },
      {
        id: "run-3",
        type: "error",
        title: "Shipping label generation failed",
        timestamp: "Tue 2025-12-30 10:19:55",
        detail: "Carrier API returned 500.",
        searchText: "shipping label failed",
      },
    ] as LogRunItem[],
  },
];

function getCounts(entries: LogEntry[]): LogCounts {
  return entries.reduce(
    (acc, entry) => {
      acc.total += 1;
      if (entry.type === "error") acc.error += 1;
      if (entry.type === "warning") acc.warning += 1;
      if (entry.type === "success") acc.success += 1;
      if (entry.runItems?.length) {
        acc.total += entry.runItems.length;
        entry.runItems.forEach((item) => {
          if (item.type === "error") acc.error += 1;
          if (item.type === "warning") acc.warning += 1;
          if (item.type === "success") acc.success += 1;
        });
      }
      return acc;
    },
    { total: 0, error: 0, warning: 0, success: 0 },
  );
}

export const Default: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);
    const [scope, setScope] = useState<LogScopeFilter>("all");
    const [filter, setFilter] = useState<LogTypeFilter>("all");
    const [searchValue, setSearchValue] = useState("");
    const [expandedRuns, setExpandedRuns] = useState<Set<string>>(() => new Set(["log-4"]));
    const wrapperRef = useRef<HTMLDivElement | null>(null);
    const [stickToBottom, setStickToBottom] = useState(true);

    const counts = useMemo(() => getCounts(sampleEntries), []);

    const filteredEntries = useMemo(() => {
      const query = searchValue.trim().toLowerCase();
      const matchesSearch = (value?: string) => !query || (value || "").toLowerCase().includes(query);

      return sampleEntries.reduce<LogEntry[]>((acc, entry) => {
        if (scope !== "all" && entry.source !== scope) {
          return acc;
        }

        if (entry.type === "run") {
          const runItems = entry.runItems || [];
          const filteredRunItems = runItems.filter((item) => {
            const typeMatch = filter === "all" || item.type === filter;
            const searchMatch =
              matchesSearch(item.searchText) || matchesSearch(typeof item.title === "string" ? item.title : undefined);
            return typeMatch && searchMatch;
          });
          const entrySearchMatch =
            matchesSearch(entry.searchText) || matchesSearch(typeof entry.title === "string" ? entry.title : undefined);

          const typeMatch = filter === "all" ? true : filteredRunItems.length > 0;
          const searchMatch = query ? entrySearchMatch || filteredRunItems.length > 0 : true;

          if (typeMatch && searchMatch) {
            acc.push({ ...entry, runItems: filteredRunItems });
          }
          return acc;
        }

        if (filter !== "all" && entry.type !== filter) {
          return acc;
        }

        const entrySearchMatch =
          matchesSearch(entry.searchText) || matchesSearch(typeof entry.title === "string" ? entry.title : undefined);
        if (!entrySearchMatch) {
          return acc;
        }

        acc.push(entry);
        return acc;
      }, []);
    }, [filter, scope, searchValue]);

    useEffect(() => {
      const wrapper = wrapperRef.current;
      if (!wrapper) {
        return;
      }

      const scrollContainer = wrapper.querySelector<HTMLDivElement>("[data-log-scroll]");
      if (!scrollContainer) {
        return;
      }

      const handleScroll = () => {
        const threshold = 16;
        const { scrollTop, scrollHeight, clientHeight } = scrollContainer;
        const isAtBottom = scrollHeight - scrollTop - clientHeight <= threshold;
        setStickToBottom(isAtBottom);
      };

      scrollContainer.addEventListener("scroll", handleScroll);
      handleScroll();

      return () => {
        scrollContainer.removeEventListener("scroll", handleScroll);
      };
    }, []);

    useEffect(() => {
      if (!stickToBottom) {
        return;
      }

      const wrapper = wrapperRef.current;
      if (!wrapper) {
        return;
      }

      const scrollContainer = wrapper.querySelector<HTMLDivElement>("[data-log-scroll]");
      if (!scrollContainer) {
        return;
      }

      scrollContainer.scrollTop = scrollContainer.scrollHeight;
    }, [filteredEntries, stickToBottom]);

    return (
      <div className="relative h-[600px] bg-slate-100" ref={wrapperRef}>
        <div className="absolute top-4 right-4 z-10">
          <Button variant="secondary" onClick={() => setIsOpen(true)}>
            Open Log Sidebar
          </Button>
        </div>
        <CanvasLogSidebar
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          filter={filter}
          onFilterChange={setFilter}
          scope={scope}
          onScopeChange={setScope}
          searchValue={searchValue}
          onSearchChange={setSearchValue}
          entries={filteredEntries}
          counts={counts}
          expandedRuns={expandedRuns}
          onToggleRun={(runId) =>
            setExpandedRuns((prev) => {
              const next = new Set(prev);
              if (next.has(runId)) {
                next.delete(runId);
              } else {
                next.add(runId);
              }
              return next;
            })
          }
        />
      </div>
    );
  },
};

export const Empty: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);
    const [scope, setScope] = useState<LogScopeFilter>("all");
    const [filter, setFilter] = useState<LogTypeFilter>("all");
    const [searchValue, setSearchValue] = useState("");

    return (
      <div className="relative h-[600px] bg-slate-100">
        <CanvasLogSidebar
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          filter={filter}
          onFilterChange={setFilter}
          scope={scope}
          onScopeChange={setScope}
          searchValue={searchValue}
          onSearchChange={setSearchValue}
          entries={[]}
          counts={{ total: 0, error: 0, warning: 0, success: 0 }}
          expandedRuns={new Set()}
          onToggleRun={() => {}}
        />
      </div>
    );
  },
};

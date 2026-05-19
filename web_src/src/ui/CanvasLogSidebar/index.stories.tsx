import type { Meta, StoryObj } from "@storybook/react";
import { useEffect, useMemo, useRef, useState } from "react";

import { CanvasLogSidebar, type LogCounts, type LogEntry } from "./index";
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
    source: "canvas",
    searchText: "rate limit warning",
  },
  {
    id: "log-3",
    type: "error",
    title: "Webhook signature mismatch",
    timestamp: "Tue 2025-12-30 10:17:03",
    source: "canvas",
    searchText: "webhook signature mismatch",
  },
];

function getCounts(entries: LogEntry[]): LogCounts {
  return entries.reduce(
    (acc, entry) => {
      acc.total += 1;
      if (entry.type === "error") acc.error += 1;
      if (entry.type === "warning") acc.warning += 1;
      if (entry.type === "success") acc.success += 1;
      return acc;
    },
    { total: 0, error: 0, warning: 0, success: 0 },
  );
}

export const Default: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);
    const [searchValue, setSearchValue] = useState("");
    const wrapperRef = useRef<HTMLDivElement | null>(null);
    const [stickToBottom, setStickToBottom] = useState(true);

    const counts = useMemo(() => getCounts(sampleEntries), []);

    const filteredEntries = useMemo(() => {
      const query = searchValue.trim().toLowerCase();
      if (!query) return sampleEntries;
      const matchesSearch = (value?: string) => (value || "").toLowerCase().includes(query);
      return sampleEntries.filter(
        (entry) => matchesSearch(entry.searchText) || matchesSearch(typeof entry.title === "string" ? entry.title : ""),
      );
    }, [searchValue]);

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
          searchValue={searchValue}
          onSearchChange={setSearchValue}
          entries={filteredEntries}
          counts={counts}
        />
      </div>
    );
  },
};

export const Empty: Story = {
  render: () => {
    const [isOpen, setIsOpen] = useState(true);
    const [searchValue, setSearchValue] = useState("");

    return (
      <div className="relative h-[600px] bg-slate-100">
        <CanvasLogSidebar
          isOpen={isOpen}
          onClose={() => setIsOpen(false)}
          searchValue={searchValue}
          onSearchChange={setSearchValue}
          entries={[]}
          counts={{ total: 0, error: 0, warning: 0, success: 0 }}
        />
      </div>
    );
  },
};

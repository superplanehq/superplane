import { Plus, Search } from "lucide-react";
import { useEffect, useMemo, useState } from "react";

import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogDescription, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { IntegrationIcon } from "@/ui/componentSidebar/integrationIcons";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";
import { CUSTOM_MCP_CATALOG_ID, filterMCPCatalog, groupMCPCatalog, MCP_CATALOG, type MCPCatalogEntry } from "./mcpCatalog";

export function MCPAddPicker({
  open,
  onClose,
  onSelect,
}: {
  open: boolean;
  onClose: () => void;
  onSelect: (entry?: MCPCatalogEntry) => void;
}) {
  const [query, setQuery] = useState("");
  const entries = useMemo(() => filterMCPCatalog(MCP_CATALOG, query), [query]);

  useEffect(() => {
    if (open) {
      setQuery("");
    }
  }, [open]);

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && onClose()}>
      <DialogContent
        className="flex max-h-[min(42rem,85vh)] max-w-lg flex-col gap-3 sm:max-w-2xl"
        data-testid="mcp-add-picker"
      >
        <DialogHeader>
          <DialogTitle>{AGENT_RESOURCES_COPY.addConnection}</DialogTitle>
          <DialogDescription>{AGENT_RESOURCES_COPY.catalogDescription}</DialogDescription>
        </DialogHeader>
        <div className="relative">
          <Search
            className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-muted-foreground"
            aria-hidden
          />
          <Input
            id="mcp-catalog-search"
            value={query}
            onChange={(event) => setQuery(event.target.value)}
            placeholder={AGENT_RESOURCES_COPY.catalogSearchPlaceholder}
            aria-label={AGENT_RESOURCES_COPY.catalogSearchPlaceholder}
            className="h-9 pl-8 text-[13px] shadow-none"
            data-testid="mcp-catalog-search"
            autoFocus
          />
        </div>
        <MCPCatalogList entries={entries} onSelect={onSelect} />
        <div className="border-t border-border pt-2">
          <Button
            type="button"
            variant="ghost"
            className="h-auto w-full justify-start gap-2 px-3 py-2"
            onClick={() => onSelect(undefined)}
            data-testid={`mcp-catalog-${CUSTOM_MCP_CATALOG_ID}`}
          >
            <Plus className="size-4" aria-hidden />
            <span>{AGENT_RESOURCES_COPY.catalogCustom}</span>
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function MCPCatalogList({
  entries,
  onSelect,
}: {
  entries: MCPCatalogEntry[];
  onSelect: (entry?: MCPCatalogEntry) => void;
}) {
  const groups = useMemo(() => groupMCPCatalog(entries), [entries]);
  if (entries.length === 0) {
    return (
      <p className="min-h-0 flex-1 px-2 py-8 text-center text-[13px] text-muted-foreground">
        {AGENT_RESOURCES_COPY.catalogSearchEmpty}
      </p>
    );
  }
  return (
    <div className="min-h-0 flex-1 space-y-4 overflow-y-auto pr-1" data-testid="mcp-catalog-list">
      {groups.map((group) => (
        <section key={group.id} data-testid={`mcp-catalog-category-${group.id}`}>
          <h3 className="px-1 pb-1 text-[12px] font-medium text-muted-foreground">{group.label}</h3>
          <ul className="grid grid-cols-2 gap-1 sm:grid-cols-3">
            {group.entries.map((entry) => (
              <li key={entry.id}>
                <Button
                  type="button"
                  variant="ghost"
                  className="h-auto w-full justify-start gap-2 px-3 py-2"
                  onClick={() => onSelect(entry)}
                  data-testid={`mcp-catalog-${entry.id}`}
                >
                  <IntegrationIcon integrationName={entry.icon} iconSlug={entry.icon} className="size-4" />
                  <span className="truncate">{entry.label}</span>
                </Button>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </div>
  );
}

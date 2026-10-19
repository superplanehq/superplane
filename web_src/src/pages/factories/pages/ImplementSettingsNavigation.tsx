import type { SuperplaneComponentsNode } from "@/api-client";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { useEffect, useRef } from "react";

export function ImplementSettingsNavigation({
  nodes,
  selectedNodeId,
  onNodeSelect,
}: {
  nodes: SuperplaneComponentsNode[];
  selectedNodeId: string;
  onNodeSelect: (nodeId: string) => void;
}) {
  const activeItemRef = useRef<HTMLButtonElement | null>(null);
  useEffect(() => {
    activeItemRef.current?.scrollIntoView?.({ block: "nearest", inline: "nearest" });
  }, [selectedNodeId]);
  return (
    <nav
      aria-label="Automation settings"
      className="shrink-0 overflow-x-auto border-b border-border px-3 py-3 md:w-56 md:overflow-y-auto md:border-r md:border-b-0 md:py-6"
    >
      <ul className="flex gap-1 md:flex-col md:gap-0.5">
        {nodes.map((node) => {
          const nodeId = node.id;
          if (!nodeId) return null;
          return (
            <li key={nodeId} className="shrink-0">
              <Button
                type="button"
                variant="ghost"
                ref={nodeId === selectedNodeId ? activeItemRef : undefined}
                aria-current={nodeId === selectedNodeId ? "page" : undefined}
                onClick={() => onNodeSelect(nodeId)}
                className={cn(
                  "h-auto min-h-9 justify-start rounded-md px-2.5 py-2 text-left text-[13px] font-normal tracking-[-0.01em] text-foreground/70 hover:bg-sidebar-accent hover:text-foreground md:w-full md:whitespace-normal",
                  nodeId === selectedNodeId && "bg-sidebar-accent font-medium text-foreground",
                )}
              >
                {node.name?.trim() || "Step"}
              </Button>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}

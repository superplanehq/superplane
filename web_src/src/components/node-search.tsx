import { useCallback, useState, useEffect } from "react";
import { Search } from "lucide-react";

import { BuiltInEdge, useReactFlow, type Node, type PanelProps } from "@xyflow/react";

import { CommandDialog, CommandGroup, CommandInput, CommandItem, CommandList } from "@/components/ui/command";
import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { resolveIcon } from "@/lib/utils";

export interface NodeSearchProps extends Omit<PanelProps, "children"> {
  // The function to search for nodes, should return an array of nodes that match the search string
  // By default, it will check for lowercase string inclusion.
  onSearch?: (searchString: string) => Node[];
  // The function to select a node, should set the node as selected and fit the view to the node
  // By default, it will set the node as selected and fit the view to the node.
  onSelectNode?: (node: Node) => void;
  open?: boolean;
  onOpenChange?: (open: boolean) => void;
}

// Helper function to extract icon slug from node data
function getNodeIconSlug(node: Node): string {
  const nodeType = node.data?.type as string | undefined;
  const isAnnotationNode = nodeType === "annotation";

  if (isAnnotationNode) {
    return "sticky-note";
  }

  // Try to get icon from component/trigger/composite data
  if (nodeType === "component" && node.data.component) {
    return (node.data.component as { iconSlug?: string }).iconSlug || "box";
  }

  if (nodeType === "trigger" && node.data.trigger) {
    return (node.data.trigger as { iconSlug?: string }).iconSlug || "play";
  }

  if (nodeType === "composite" && node.data.composite) {
    return (node.data.composite as { iconSlug?: string }).iconSlug || "boxes";
  }

  // Default fallback
  return "box";
}

export function NodeSearchInternal({ onSearch, onSelectNode, open, onOpenChange }: NodeSearchProps) {
  const [searchResults, setSearchResults] = useState<Node[]>([]);
  const [searchString, setSearchString] = useState<string>("");
  const { getNodes, fitView, setNodes } = useReactFlow<Node, BuiltInEdge>();

  const defaultOnSearch = useCallback(
    (searchString: string) => {
      const nodes = getNodes();
      return nodes.filter((node) => (node.data.label as string).toLowerCase().includes(searchString.toLowerCase()));
    },
    [getNodes],
  );

  const onChange = useCallback(
    (searchString: string) => {
      setSearchString(searchString);
      if (searchString.length > 0) {
        onOpenChange?.(true);
        const results = (onSearch || defaultOnSearch)(searchString);
        setSearchResults(results);
      }
    },
    [onSearch, onOpenChange],
  );

  const defaultOnSelectNode = useCallback(
    (node: Node) => {
      setNodes((nodes) => nodes.map((n) => (n.id === node.id ? { ...n, selected: true } : n)));
      fitView({ nodes: [node], duration: 500 });
    },
    [fitView, setNodes],
  );

  const onSelect = useCallback(
    (node: Node) => {
      // Always call the default behavior (select node + fit view)
      defaultOnSelectNode(node);
      // Then call custom handler if provided (e.g., open sidebar)
      onSelectNode?.(node);
      setSearchString("");
      onOpenChange?.(false);
    },
    [onSelectNode, defaultOnSelectNode, onOpenChange],
  );

  return (
    <>
      <CommandInput
        placeholder="Search components..."
        onValueChange={onChange}
        value={searchString}
        onFocus={() => onOpenChange?.(true)}
      />

      {open && (
        <CommandList>
          {searchResults.length === 0 ? null : (
            <CommandGroup heading="Components">
              {searchResults.map((node) => {
                const isAnnotationNode = (node.data as { type?: string })?.type === "annotation";
                const fallbackLabel =
                  (node.data as { nodeName?: string })?.nodeName || (node.data as { label?: string })?.label;
                const displayLabel = isAnnotationNode ? "Note" : fallbackLabel || node.id;
                const iconSlug = getNodeIconSlug(node);
                const IconComponent = resolveIcon(iconSlug);

                return (
                  <CommandItem key={node.id} onSelect={() => onSelect(node)}>
                    <div className="flex items-center gap-2 w-full min-w-0">
                      <IconComponent className="h-4 w-4 shrink-0 text-muted-foreground" />
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="truncate">{displayLabel}</span>
                        </TooltipTrigger>
                        {displayLabel.length > 40 && <TooltipContent>{displayLabel}</TooltipContent>}
                      </Tooltip>
                      <Tooltip>
                        <TooltipTrigger asChild>
                          <span className="text-muted-foreground text-xs ml-auto truncate max-w-[200px] shrink-0">
                            {node.id}
                          </span>
                        </TooltipTrigger>
                        {node.id.length > 25 && <TooltipContent>{node.id}</TooltipContent>}
                      </Tooltip>
                    </div>
                  </CommandItem>
                );
              })}
            </CommandGroup>
          )}
        </CommandList>
      )}
    </>
  );
}

export function NodeSearch({ onSearch, onSelectNode }: NodeSearchProps) {
  const [open, setOpen] = useState(false);

  // Add keyboard shortcut Ctrl+K / Cmd+K
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key === "k") {
        e.preventDefault();
        setOpen(true);
      }
    };
    document.addEventListener("keydown", handleKeyDown);
    return () => document.removeEventListener("keydown", handleKeyDown);
  }, []);

  return (
    <>
      <Tooltip>
        <TooltipTrigger asChild>
          <Button variant="ghost" size="icon-sm" onClick={() => setOpen(true)}>
            <Search className="h-3 w-3" />
          </Button>
        </TooltipTrigger>
        <TooltipContent>Search components (Ctrl/Cmd + K)</TooltipContent>
      </Tooltip>
      <CommandDialog open={open} onOpenChange={setOpen} className="max-w-xl sm:max-w-2xl md:max-w-3xl">
        <NodeSearchInternal onSearch={onSearch} onSelectNode={onSelectNode} open={open} onOpenChange={setOpen} />
      </CommandDialog>
    </>
  );
}

export interface NodeSearchDialogProps extends NodeSearchProps {
  title?: string;
}

export function NodeSearchDialog({ onSearch, onSelectNode, open, onOpenChange }: NodeSearchDialogProps) {
  return (
    <CommandDialog open={open} onOpenChange={onOpenChange} className="max-w-xl sm:max-w-2xl md:max-w-3xl">
      <NodeSearchInternal onSearch={onSearch} onSelectNode={onSelectNode} open={open} onOpenChange={onOpenChange} />
    </CommandDialog>
  );
}

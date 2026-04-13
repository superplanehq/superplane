import { Button } from "@/components/ui/button";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { MermaidDiagram } from "@/components/MermaidDiagram";
import { MermaidDiagramDialog } from "@/components/MermaidDiagramDialog";
import { type DiffItem, summarizeProposalDiff, proposalToMermaid } from "@/lib/proposalDiagram";
import { getHeaderIconSrc } from "@/ui/componentSidebar/integrationIcons";
import type { AiBuilderProposal } from "@/ui/BuildingBlocksSidebar/agentChat";
import { ArrowRight, Box, Maximize2, Minus, Pencil, Plus, Unlink } from "lucide-react";
import { useMemo, useState } from "react";

export type ProposalsListProps = {
  pendingProposal: AiBuilderProposal;
  applyShortcutHint: string;
  onApplyProposal: () => void;
  onDiscardProposal: () => void;
  isApplyingProposal: boolean;
  aiError: string | null;
  disabled: boolean;
};

function DiffItemRow({ item }: { item: DiffItem }) {
  const iconSrc = item.blockName ? getHeaderIconSrc(item.blockName) : undefined;

  return (
    <div className="flex items-center gap-1.5 py-0.5">
      {iconSrc ? (
        <img src={iconSrc} alt="" className="h-3.5 w-3.5 object-contain shrink-0" />
      ) : (
        <Box className="h-3.5 w-3.5 shrink-0 opacity-50" />
      )}
      <span>{item.label}</span>
    </div>
  );
}

function DiffBadge({
  icon: Icon,
  count,
  noun,
  items,
  color,
}: {
  icon: React.ComponentType<{ className?: string }>;
  count: number;
  noun: string;
  items: DiffItem[];
  color: "green" | "amber" | "red";
}) {
  if (count === 0) {
    return null;
  }

  const colorClasses = {
    green: "text-emerald-700 bg-emerald-50 border-emerald-200",
    amber: "text-amber-700 bg-amber-50 border-amber-200",
    red: "text-red-700 bg-red-50 border-red-200",
  };

  const badge = (
    <span
      className={`inline-flex items-center gap-1 rounded-md border px-1.5 py-0.5 text-[11px] font-medium ${colorClasses[color]}`}
    >
      <Icon className="h-3 w-3" />
      {count} {count === 1 ? noun : `${noun}s`}
    </span>
  );

  return (
    <Tooltip>
      <TooltipTrigger asChild>{badge}</TooltipTrigger>
      <TooltipContent side="top" className="max-w-[220px]">
        {items.map((item, i) => (
          <DiffItemRow key={i} item={item} />
        ))}
      </TooltipContent>
    </Tooltip>
  );
}

export function ProposalsList({
  pendingProposal,
  applyShortcutHint,
  onApplyProposal,
  onDiscardProposal,
  isApplyingProposal,
  aiError,
  disabled,
}: ProposalsListProps) {
  const isDisabled = disabled || isApplyingProposal || pendingProposal.operations.length === 0;
  const [diagramOpen, setDiagramOpen] = useState(false);

  const diff = useMemo(() => summarizeProposalDiff(pendingProposal.operations), [pendingProposal.operations]);

  const showDiagram = diff.addedNodes.length + diff.removedNodes.length >= 2 || diff.addedConnections.length > 0;

  const previewDiagram = useMemo(
    () => (showDiagram ? proposalToMermaid(pendingProposal.operations, "preview") : ""),
    [pendingProposal.operations, showDiagram],
  );
  const expandedDiagram = useMemo(
    () => (showDiagram ? proposalToMermaid(pendingProposal.operations, "expanded") : ""),
    [pendingProposal.operations, showDiagram],
  );

  return (
    <>
      <div className="rounded-lg border border-blue-200 bg-blue-50/60 overflow-hidden">
        <div className="px-4 pt-3 pb-2 space-y-1.5">
          <p className="text-sm font-medium text-blue-900">{pendingProposal.summary}</p>

          <div className="flex flex-wrap items-center gap-1.5">
            <DiffBadge
              icon={Plus}
              count={diff.addedNodes.length}
              noun="component"
              items={diff.addedNodes}
              color="green"
            />
            <DiffBadge
              icon={Pencil}
              count={diff.modifiedNodes.length}
              noun="modified"
              items={diff.modifiedNodes}
              color="amber"
            />
            <DiffBadge
              icon={Minus}
              count={diff.removedNodes.length}
              noun="removed"
              items={diff.removedNodes}
              color="red"
            />
            <DiffBadge
              icon={ArrowRight}
              count={diff.addedConnections.length}
              noun="connection"
              items={diff.addedConnections}
              color="green"
            />
            <DiffBadge
              icon={Unlink}
              count={diff.removedConnections.length}
              noun="disconnection"
              items={diff.removedConnections}
              color="red"
            />
          </div>
        </div>

        {showDiagram ? (
          <button
            type="button"
            onClick={() => setDiagramOpen(true)}
            disabled={isDisabled}
            className="group relative block w-full border-t border-b border-blue-100 bg-white/60 cursor-pointer hover:bg-white/80 transition-colors disabled:cursor-default disabled:opacity-60"
          >
            <div className="max-h-[140px] overflow-hidden px-2 py-2">
              <MermaidDiagram
                definition={previewDiagram}
                className="pointer-events-none [&_svg]:max-w-full [&_svg]:h-auto [&_svg]:mx-auto"
              />
            </div>
            <div className="absolute inset-x-0 bottom-0 h-8 bg-gradient-to-t from-white/90 to-transparent" />
            <span className="absolute right-2 bottom-2 inline-flex items-center gap-1 rounded bg-blue-100 px-1.5 py-0.5 text-[10px] text-blue-700 opacity-0 group-hover:opacity-100 transition-opacity">
              <Maximize2 className="h-2.5 w-2.5" />
              Expand
            </span>
          </button>
        ) : null}

        <div className="px-4 py-3 flex items-center gap-2">
          <Button size="sm" onClick={onApplyProposal} disabled={isDisabled}>
            Apply changes
          </Button>
          <Button size="sm" variant="outline" onClick={onDiscardProposal} disabled={isDisabled}>
            Discard
          </Button>
          <span className="ml-auto text-[10px] text-blue-600">{applyShortcutHint} to accept</span>
        </div>

        {aiError ? <p className="text-xs text-red-700 px-4 pb-3 -mt-1">{aiError}</p> : null}
      </div>

      {showDiagram ? (
        <MermaidDiagramDialog open={diagramOpen} onOpenChange={setDiagramOpen} definition={expandedDiagram} />
      ) : null}
    </>
  );
}

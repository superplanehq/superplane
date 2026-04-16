import { Button } from "@/components/ui/button";
import { useMemo } from "react";
import type { AiBuilderProposal } from "./agentChat";
import { useFormatOperation } from "./useFormatOperation";

export type ProposalsListProps = {
  pendingProposal: AiBuilderProposal;
  applyShortcutHint: string;
  onApplyProposal: () => void;
  onDiscardProposal: () => void;
  isApplyingProposal: boolean;
  aiError: string | null;
  disabled: boolean;
};

export function ProposalsList({
  pendingProposal,
  applyShortcutHint,
  onApplyProposal,
  onDiscardProposal,
  isApplyingProposal,
  aiError,
  disabled,
}: ProposalsListProps) {
  const formatOperation = useFormatOperation();
  const pendingProposalSummaries = useMemo(
    () => (pendingProposal.changeset.changes || []).map((change) => formatOperation(change)),
    [formatOperation, pendingProposal],
  );

  const isDisabled = disabled || isApplyingProposal || (pendingProposal.changeset.changes || []).length === 0;

  return (
    <div className="relative rounded-md border border-blue-200 bg-blue-50 px-3 py-3 space-y-2">
      <span className="absolute right-2 top-2 text-[10px] text-blue-800">{`${applyShortcutHint} to accept`}</span>
      <ul className="text-sm text-blue-900 list-disc pl-5 space-y-1">
        {pendingProposalSummaries.map((summary, index) => (
          <li key={`${pendingProposal.id}-${index}`}>{summary}</li>
        ))}
      </ul>

      <div className="flex items-center gap-2 pt-1">
        <Button size="sm" onClick={onApplyProposal} disabled={isDisabled}>
          Apply changes
        </Button>
        <Button size="sm" variant="outline" onClick={onDiscardProposal} disabled={isDisabled}>
          Discard
        </Button>
      </div>

      {aiError ? <p className="text-xs text-red-700">{aiError}</p> : null}
    </div>
  );
}

import type { CanvasesCanvasVersion } from "@/api-client";
import { useMemo, useState } from "react";
import { draftBranchName, draftDisplayName, draftOwnerName } from "@/lib/draftVersion";
import { StartEditingDraftListView } from "./StartEditingDraftListView";
import { StartEditingMenuHome } from "./StartEditingMenuHome";

type StartEditingDraftPickerProps = {
  drafts: CanvasesCanvasVersion[];
  defaultDraft: CanvasesCanvasVersion | null;
  isSubmitting?: boolean;
  onContinueDraft: (branchName: string) => void;
  onCreateDraft: () => void;
  onClose: () => void;
};

export function StartEditingDraftPicker({
  drafts,
  defaultDraft,
  isSubmitting,
  onContinueDraft,
  onCreateDraft,
  onClose,
}: StartEditingDraftPickerProps) {
  const [showList, setShowList] = useState(false);
  const [searchQuery, setSearchQuery] = useState("");
  const filteredDrafts = useMemo(() => {
    const query = searchQuery.trim().toLowerCase();
    if (!query) {
      return drafts;
    }

    return drafts.filter((draft) => {
      const displayName = draftDisplayName(draft).toLowerCase();
      const owner = draftOwnerName(draft).toLowerCase();
      return (
        displayName.includes(query) || owner.includes(query) || draftBranchName(draft).toLowerCase().includes(query)
      );
    });
  }, [drafts, searchQuery]);
  const continueDraft = defaultDraft ?? drafts[0] ?? null;
  const continueBranchName = continueDraft ? draftBranchName(continueDraft) : "";

  const closeAndRun = (action: () => void) => {
    action();
    onClose();
  };

  if (!showList) {
    return (
      <StartEditingMenuHome
        drafts={drafts}
        continueDraft={continueDraft}
        continueBranchName={continueBranchName}
        isSubmitting={isSubmitting}
        onContinueDraft={(branchName) => closeAndRun(() => onContinueDraft(branchName))}
        onCreateDraft={() => closeAndRun(onCreateDraft)}
        onShowList={() => setShowList(true)}
      />
    );
  }

  return (
    <StartEditingDraftListView
      filteredDrafts={filteredDrafts}
      continueBranchName={continueBranchName}
      searchQuery={searchQuery}
      onSearchQueryChange={setSearchQuery}
      onContinueDraft={(branchName) => closeAndRun(() => onContinueDraft(branchName))}
      onBack={() => setShowList(false)}
    />
  );
}

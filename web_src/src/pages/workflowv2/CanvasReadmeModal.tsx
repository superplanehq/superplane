import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";
import { Input } from "@/components/ui/input";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import {
  CanvasMarkdown,
  type NodeChipDetails,
  type NodeChipIcon,
} from "@/ui/Markdown/CanvasMarkdown";
import { useEffect, useMemo, useState } from "react";

//
// CanvasReadmeModal is a near-full-screen dialog that mirrors the canvas's
// Live / Editor split: opening it from the Live tab shows the published
// readme read-only; opening it from the Editor tab shows a draft editor with
// Save / Request-change actions in a sticky footer. Mode is controlled by the
// caller — there is no internal mode toggle.
//

export type CanvasReadmeModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: "live" | "edit";
  changeManagementEnabled: boolean;
  liveContent: string;
  draftContent: string;
  isLoadingLive: boolean;
  isLoadingDraft: boolean;
  isSavingDraft: boolean;
  isCreatingChangeRequest: boolean;
  nodes: Record<string, string>;
  icons?: Record<string, NodeChipIcon>;
  details?: Record<string, NodeChipDetails>;
  linkFor: (slug: string) => string;
  onSaveDraft: (content: string) => Promise<void>;
  onCreateChangeRequest: (args: { title: string; description: string }) => Promise<void>;
};

type NodeRefs = {
  nodes: Record<string, string>;
  icons?: Record<string, NodeChipIcon>;
  details?: Record<string, NodeChipDetails>;
  linkFor: (slug: string) => string;
};

export function CanvasReadmeModal(props: CanvasReadmeModalProps) {
  const {
    open,
    onOpenChange,
    mode,
    changeManagementEnabled,
    liveContent,
    draftContent,
    isLoadingLive,
    isLoadingDraft,
    isSavingDraft,
    isCreatingChangeRequest,
    nodes,
    icons,
    details,
    linkFor,
    onSaveDraft,
    onCreateChangeRequest,
  } = props;

  const nodeRefs = useMemo(
    () => ({ nodes, icons, details, linkFor }),
    [nodes, icons, details, linkFor],
  );
  const subtitle = mode === "edit" ? "Draft" : "Published";

  const [editorValue, setEditorValue] = useState<string>("");
  const [initialValue, setInitialValue] = useState<string>("");
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [isCRModalOpen, setIsCRModalOpen] = useState(false);
  const [crTitle, setCrTitle] = useState("Update README");
  const [crDescription, setCrDescription] = useState("");

  const isDirty = mode === "edit" && editorValue !== initialValue;

  //
  // Seed the editor once the draft has finished loading (edit mode only).
  // When the caller's draft is empty we fall back to the live content so
  // authors start from the current published readme. We reseed each time the
  // modal reopens, so closing and reopening restores the latest persisted
  // draft rather than a stale in-memory value.
  //
  useEffect(() => {
    if (mode !== "edit") return;
    if (!open) return;
    if (isLoadingDraft || isLoadingLive) return;
    const seed = draftContent || liveContent;
    setEditorValue(seed);
    setInitialValue(seed);
  }, [mode, open, isLoadingDraft, isLoadingLive, draftContent, liveContent]);

  //
  // Intercept overlay / Esc / close-button dismissals when the draft has
  // unsaved edits. Radix routes all of those through onOpenChange(false) so
  // a single guard covers every dismissal path.
  //
  const handleOpenChange = (nextOpen: boolean) => {
    if (!nextOpen && isDirty) {
      if (!window.confirm("Discard unsaved readme changes?")) {
        return;
      }
    }
    onOpenChange(nextOpen);
  };

  //
  // beforeunload guard active only when the modal is open and dirty — this
  // catches hard refresh / tab close / external navigations that would blow
  // past Radix's close lifecycle.
  //
  useEffect(() => {
    if (!open || !isDirty) return;
    const handler = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", handler);
    return () => window.removeEventListener("beforeunload", handler);
  }, [open, isDirty]);

  const handleSaveDraft = async () => {
    await onSaveDraft(editorValue);
    setInitialValue(editorValue);
    setStatusMessage("Draft saved.");
    window.setTimeout(() => setStatusMessage(null), 2500);
  };

  const handleOpenCR = async () => {
    if (isDirty) {
      await onSaveDraft(editorValue);
      setInitialValue(editorValue);
    }
    setIsCRModalOpen(true);
  };

  const handleConfirmCR = async () => {
    await onCreateChangeRequest({
      title: crTitle.trim() || "Update README",
      description: crDescription,
    });
    setIsCRModalOpen(false);
  };

  return (
    <>
      <Dialog open={open} onOpenChange={handleOpenChange}>
        <DialogContent size="large" className="flex max-h-[90vh] w-[90vw] h-full flex-col gap-0 overflow-hidden p-0">
          <div className="flex h-full min-h-0 flex-col">
            <div className="flex shrink-0 items-center justify-between border-b border-gray-200 bg-white px-4 py-3">
              <div className="flex items-baseline gap-2">
                <span className="font-mono text-sm text-gray-600">Canvas Readme</span>
                <span className="text-xs uppercase tracking-wide text-gray-400">{subtitle}</span>
              </div>
            </div>

            <div className="flex min-h-0 flex-1 flex-col overflow-hidden bg-slate-50">
              {mode === "live" ? (
                <LiveReadmeBody
                  liveContent={liveContent}
                  isLoadingLive={isLoadingLive}
                  nodeRefs={nodeRefs}
                />
              ) : (
                <EditReadmeBody
                  editorValue={editorValue}
                  setEditorValue={setEditorValue}
                  statusMessage={statusMessage}
                  isDirty={isDirty}
                  changeManagementEnabled={changeManagementEnabled}
                  isLoadingLive={isLoadingLive}
                  isLoadingDraft={isLoadingDraft}
                  isSavingDraft={isSavingDraft}
                  nodeRefs={nodeRefs}
                  onSaveDraft={handleSaveDraft}
                  onRequestChange={handleOpenCR}
                />
              )}
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <Dialog open={isCRModalOpen} onOpenChange={setIsCRModalOpen}>
        <DialogContent className="max-w-xl">
          <DialogHeader>
            <DialogTitle>Request README update</DialogTitle>
            <DialogDescription>
              Your latest draft will be attached to a new change request. Approvers can review and publish it.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-3">
            <div className="space-y-1">
              <Label htmlFor="cr-title">Title</Label>
              <Input id="cr-title" value={crTitle} onChange={(e) => setCrTitle(e.target.value)} />
            </div>
            <div className="space-y-1">
              <Label htmlFor="cr-description">Description</Label>
              <Textarea
                id="cr-description"
                rows={4}
                value={crDescription}
                onChange={(e) => setCrDescription(e.target.value)}
                placeholder="Optional notes for reviewers"
              />
            </div>
          </div>

          <DialogFooter>
            <Button variant="ghost" onClick={() => setIsCRModalOpen(false)}>
              Cancel
            </Button>
            <LoadingButton onClick={handleConfirmCR} loading={isCreatingChangeRequest}>
              Create change request
            </LoadingButton>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </>
  );
}

function LiveReadmeBody({
  liveContent,
  isLoadingLive,
  nodeRefs,
}: {
  liveContent: string;
  isLoadingLive: boolean;
  nodeRefs: NodeRefs;
}) {
  const hasContent = liveContent.trim().length > 0;

  return (
    <div className="min-h-0 w-full min-w-0 flex-1 overflow-auto">
      <div className="mx-auto max-w-[1100px] px-6 py-6">
        {isLoadingLive ? (
          <p className="text-sm text-slate-500">Loading…</p>
        ) : hasContent ? (
          <CanvasMarkdown nodeRefs={nodeRefs}>{liveContent}</CanvasMarkdown>
        ) : (
          <p className="text-sm text-slate-500">No readme has been published for this canvas yet.</p>
        )}
      </div>
    </div>
  );
}

function EditReadmeBody({
  editorValue,
  setEditorValue,
  statusMessage,
  isDirty,
  changeManagementEnabled,
  isLoadingLive,
  isLoadingDraft,
  isSavingDraft,
  nodeRefs,
  onSaveDraft,
  onRequestChange,
}: {
  editorValue: string;
  setEditorValue: (value: string) => void;
  statusMessage: string | null;
  isDirty: boolean;
  changeManagementEnabled: boolean;
  isLoadingLive: boolean;
  isLoadingDraft: boolean;
  isSavingDraft: boolean;
  nodeRefs: NodeRefs;
  onSaveDraft: () => Promise<void>;
  onRequestChange: () => Promise<void>;
}) {
  return (
    <div className="flex min-h-0 flex-1 flex-col">
      {statusMessage && (
        <div className="border-b border-emerald-200 bg-emerald-50 px-4 py-2 text-sm text-emerald-800">
          {statusMessage}
        </div>
      )}

      <div className="grid min-h-0 flex-1 grid-cols-1 gap-3 overflow-hidden p-3 lg:grid-cols-2">
        <div className="flex min-h-0 flex-col rounded-md border border-slate-200 bg-white">
          <div className="border-b border-slate-200 px-3 py-2 text-xs font-medium uppercase tracking-wide text-slate-500">
            Markdown draft
          </div>
          <Textarea
            className="min-h-0 flex-1 resize-none rounded-none border-0 font-mono text-sm focus-visible:ring-0"
            value={editorValue}
            onChange={(e) => setEditorValue(e.target.value)}
            placeholder="# Canvas README&#10;&#10;Describe this canvas, link nodes with @node-name or [[node:name]]."
            disabled={isLoadingDraft || isLoadingLive}
            aria-label="Canvas readme markdown editor"
          />
        </div>

        <div className="flex min-h-0 flex-col rounded-md border border-slate-200 bg-white">
          <div className="border-b border-slate-200 px-3 py-2 text-xs font-medium uppercase tracking-wide text-slate-500">
            Preview
          </div>
          <div className="flex-1 overflow-auto p-6">
            {editorValue.trim().length > 0 ? (
              <CanvasMarkdown nodeRefs={nodeRefs}>{editorValue}</CanvasMarkdown>
            ) : (
              <p className="text-sm text-slate-500">Your draft readme is empty. Start typing on the left.</p>
            )}
          </div>
        </div>
      </div>

      <div className="flex shrink-0 items-center justify-between gap-3 border-t border-slate-200 bg-white px-4 py-3">
        <span className="text-xs text-slate-500">
          {isDirty ? "Unsaved changes" : "No unsaved changes"}
        </span>
        <div className="flex items-center gap-2">
          {changeManagementEnabled && (
            <Button size="sm" variant="default" onClick={onRequestChange}>
              Request change
            </Button>
          )}
          <LoadingButton size="sm" onClick={onSaveDraft} disabled={!isDirty} loading={isSavingDraft}>
            Save draft
          </LoadingButton>
        </div>
      </div>
    </div>
  );
}

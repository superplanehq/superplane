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
import { CanvasMarkdown } from "@/ui/Markdown/CanvasMarkdown";
import { ArrowLeft } from "lucide-react";
import { useEffect, useMemo, useState } from "react";
import { Link } from "react-router-dom";

type ReadmeViewProps = {
  mode: "live" | "edit";
  backToCanvasHref: string;
  canvasName: string;
  changeManagementEnabled: boolean;
  liveContent: string;
  draftContent: string;
  isLoadingLive: boolean;
  isLoadingDraft: boolean;
  isSavingDraft: boolean;
  isCreatingChangeRequest: boolean;
  draftVersionId?: string;
  nodes: Record<string, string>;
  linkFor: (slug: string) => string;
  onSaveDraft: (content: string) => Promise<void>;
  onCreateChangeRequest: (args: { title: string; description: string }) => Promise<void>;
};

export function ReadmeView(props: ReadmeViewProps) {
  const {
    mode,
    backToCanvasHref,
    changeManagementEnabled,
    liveContent,
    draftContent,
    isLoadingLive,
    isLoadingDraft,
    isSavingDraft,
    isCreatingChangeRequest,
    nodes,
    linkFor,
    onSaveDraft,
    onCreateChangeRequest,
  } = props;

  const nodeRefs = useMemo(() => ({ nodes, linkFor }), [nodes, linkFor]);

  if (mode === "live") {
    return (
      <LiveReadme
        backToCanvasHref={backToCanvasHref}
        liveContent={liveContent}
        isLoadingLive={isLoadingLive}
        nodeRefs={nodeRefs}
      />
    );
  }

  return (
    <EditReadme
      backToCanvasHref={backToCanvasHref}
      changeManagementEnabled={changeManagementEnabled}
      liveContent={liveContent}
      draftContent={draftContent}
      isLoadingLive={isLoadingLive}
      isLoadingDraft={isLoadingDraft}
      isSavingDraft={isSavingDraft}
      isCreatingChangeRequest={isCreatingChangeRequest}
      nodeRefs={nodeRefs}
      onSaveDraft={onSaveDraft}
      onCreateChangeRequest={onCreateChangeRequest}
    />
  );
}

type NodeRefs = { nodes: Record<string, string>; linkFor: (slug: string) => string };

function LiveReadme({
  backToCanvasHref,
  liveContent,
  isLoadingLive,
  nodeRefs,
}: {
  backToCanvasHref: string;
  liveContent: string;
  isLoadingLive: boolean;
  nodeRefs: NodeRefs;
}) {
  const hasContent = liveContent.trim().length > 0;

  return (
    <div className="mx-auto flex min-h-full max-w-[1400px] flex-col px-4 py-6">
      <div className="mb-4 flex items-center justify-between gap-3">
        <Button asChild variant="ghost" size="sm">
          <Link to={backToCanvasHref}>
            <ArrowLeft className="mr-1 h-4 w-4" />
            Back to canvas
          </Link>
        </Button>
      </div>

      <div className="flex min-h-[60vh] flex-col rounded-md border border-slate-200 bg-white">
        <div className="flex-1 overflow-auto p-6">
          {isLoadingLive ? (
            <p className="text-sm text-slate-500">Loading…</p>
          ) : hasContent ? (
            <CanvasMarkdown nodeRefs={nodeRefs}>{liveContent}</CanvasMarkdown>
          ) : (
            <p className="text-sm text-slate-500">No readme has been published for this canvas yet.</p>
          )}
        </div>
      </div>
    </div>
  );
}

function EditReadme({
  backToCanvasHref,
  changeManagementEnabled,
  liveContent,
  draftContent,
  isLoadingLive,
  isLoadingDraft,
  isSavingDraft,
  isCreatingChangeRequest,
  nodeRefs,
  onSaveDraft,
  onCreateChangeRequest,
}: {
  backToCanvasHref: string;
  changeManagementEnabled: boolean;
  liveContent: string;
  draftContent: string;
  isLoadingLive: boolean;
  isLoadingDraft: boolean;
  isSavingDraft: boolean;
  isCreatingChangeRequest: boolean;
  nodeRefs: NodeRefs;
  onSaveDraft: (content: string) => Promise<void>;
  onCreateChangeRequest: (args: { title: string; description: string }) => Promise<void>;
}) {
  const [editorValue, setEditorValue] = useState<string>("");
  const [initialValue, setInitialValue] = useState<string>("");
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [isCRModalOpen, setIsCRModalOpen] = useState(false);
  const [crTitle, setCrTitle] = useState("Update README");
  const [crDescription, setCrDescription] = useState("");

  //
  // Seed the editor once the draft has finished loading. When the caller's
  // draft is empty we fall back to the live content so they start from the
  // current published readme instead of a blank textarea.
  //
  useEffect(() => {
    if (isLoadingDraft || isLoadingLive) return;
    const seed = draftContent || liveContent;
    setEditorValue(seed);
    setInitialValue(seed);
  }, [isLoadingDraft, isLoadingLive, draftContent, liveContent]);

  const isDirty = editorValue !== initialValue;

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

  const handleBackClick = (event: React.MouseEvent<HTMLAnchorElement>) => {
    if (!isDirty) return;
    if (!window.confirm("Discard unsaved readme changes?")) {
      event.preventDefault();
    }
  };

  //
  // Intercept full-page refresh / navigation when the draft has unsaved
  // changes. Mirrors the unload guard used by the canvas editor.
  //
  useEffect(() => {
    if (!isDirty) return;
    const handler = (event: BeforeUnloadEvent) => {
      event.preventDefault();
      event.returnValue = "";
    };
    window.addEventListener("beforeunload", handler);
    return () => window.removeEventListener("beforeunload", handler);
  }, [isDirty]);

  return (
    <div className="mx-auto flex min-h-full max-w-[1400px] flex-col px-4 py-6">
      <div className="mb-4 flex items-center justify-between gap-3">
        <Button asChild variant="ghost" size="sm">
          <Link to={backToCanvasHref} onClick={handleBackClick}>
            <ArrowLeft className="mr-1 h-4 w-4" />
            Back to canvas
          </Link>
        </Button>

        <div className="flex items-center gap-2">
          <LoadingButton size="sm" onClick={handleSaveDraft} disabled={!isDirty} loading={isSavingDraft}>
            Save draft
          </LoadingButton>
          {changeManagementEnabled && (
            <Button size="sm" variant="default" onClick={handleOpenCR}>
              Request change
            </Button>
          )}
        </div>
      </div>

      {statusMessage && (
        <div className="mb-3 rounded border border-emerald-200 bg-emerald-50 px-3 py-2 text-sm text-emerald-800">
          {statusMessage}
        </div>
      )}

      <div className="grid min-h-0 flex-1 grid-cols-1 gap-4 lg:grid-cols-2">
        <div className="flex min-h-[60vh] flex-col rounded-md border border-slate-200 bg-white">
          <div className="border-b border-slate-200 px-3 py-2 text-xs font-medium uppercase tracking-wide text-slate-500">
            Markdown draft
          </div>
          <Textarea
            className="flex-1 resize-none rounded-none border-0 font-mono text-sm focus-visible:ring-0"
            value={editorValue}
            onChange={(e) => setEditorValue(e.target.value)}
            placeholder="# Canvas README&#10;&#10;Describe this canvas, link nodes with @node-name or [[node:name]]."
            disabled={isLoadingDraft || isLoadingLive}
            aria-label="Canvas readme markdown editor"
          />
        </div>

        <div className="flex min-h-[60vh] flex-col rounded-md border border-slate-200 bg-white">
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
    </div>
  );
}

import { CanvasesCanvasVersion } from "@/api-client";
import { Button } from "@/components/ui/button";
import { LoadingButton } from "@/components/ui/loading-button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/ui/accordion";
import { TriangleAlert } from "lucide-react";
import { useEffect, useState } from "react";
import { DraftNodeDiffSummary, DraftNodeDiffView } from "./draftNodeDiff";
import { NodeDiffSummaryCounts } from "./VersionNodeDiff";

interface CreateChangeRequestModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  pending: boolean;
  version?: CanvasesCanvasVersion;
  title: string;
  description: string;
  onTitleChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  diffSummary: DraftNodeDiffSummary;
  isDraftOutdated?: boolean;
  onPublish: () => void;
}

export function CreateChangeRequestModal({
  open,
  onOpenChange,
  pending,
  version,
  title,
  description,
  onTitleChange,
  onDescriptionChange,
  diffSummary,
  isDraftOutdated = false,
  onPublish,
}: CreateChangeRequestModalProps) {
  const [isDescriptionEditorOpen, setIsDescriptionEditorOpen] = useState(false);

  useEffect(() => {
    if (!open) {
      return;
    }
    setIsDescriptionEditorOpen(false);
  }, [open]);

  const handlePublish = () => {
    if (isDraftOutdated) {
      const confirmed = window.confirm(
        "This draft is outdated because the live version was updated after this draft was created. Create anyway?",
      );
      if (!confirmed) {
        return;
      }
    }

    onPublish();
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="min-w-[70vw] max-w-6xl max-h-[90vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>Create Change Request</DialogTitle>
          <DialogDescription className="text-gray-500">
            Add a title and summary. This will create a change request snapshot from your current draft.
          </DialogDescription>
        </DialogHeader>

        {!version ? (
          <div className="rounded-md border border-slate-200 bg-slate-50 p-3 text-sm text-slate-700">
            Enable edit mode and save your draft before creating a change request.
          </div>
        ) : (
          <div className="space-y-4">
            <div className="space-y-1">
              <Label htmlFor="create-change-request-title">Title</Label>
              <Input
                id="create-change-request-title"
                value={title}
                onChange={(event) => onTitleChange(event.target.value)}
                placeholder="Title"
              />
            </div>

            <div className="space-y-1">
              {isDescriptionEditorOpen ? (
                <>
                  <Label htmlFor="create-change-request-description">Description</Label>
                  <Textarea
                    id="create-change-request-description"
                    value={description}
                    onChange={(event) => onDescriptionChange(event.target.value)}
                    rows={8}
                    placeholder="Add Description..."
                  />
                </>
              ) : (
                <button
                  type="button"
                  onClick={() => setIsDescriptionEditorOpen(true)}
                  className="text-sm text-slate-500 hover:text-slate-600"
                >
                  Add Description...
                </button>
              )}
            </div>

            <section className="space-y-2">
              <p className="flex items-center text-sm leading-none font-medium text-gray-800 select-none">
                Draft Diff Summary
              </p>
              <NodeDiffSummaryCounts
                addedCount={diffSummary.addedCount}
                updatedCount={diffSummary.updatedCount}
                removedCount={diffSummary.removedCount}
              />
              {diffSummary.items.length === 0 ? (
                <p className="text-xs text-slate-600">No node-level changes detected.</p>
              ) : (
                <Accordion type="multiple" className="w-full rounded-md border border-slate-200 px-3">
                  {diffSummary.items.map((item, index) => (
                    <AccordionItem
                      key={`${item.id}-${item.changeType}-${index}`}
                      value={`${item.id}-${item.changeType}-${index}`}
                      className="border-b-0"
                    >
                      <AccordionTrigger className="py-3 hover:no-underline">
                        <div className="flex items-center gap-2 min-w-0">
                          <span
                            className={`inline-flex min-w-8 justify-center rounded px-1.5 py-0.5 text-[11px] font-semibold ${
                              item.changeType === "removed"
                                ? "bg-red-100 text-red-700"
                                : item.changeType === "added"
                                  ? "bg-emerald-100 text-emerald-700"
                                  : "bg-sky-100 text-sky-700"
                            }`}
                          >
                            {item.changeType === "updated" ? "+/-" : item.changeType === "removed" ? "-" : "+"}
                          </span>
                          <span className="truncate text-sm text-slate-900">{item.name}</span>
                          <span className="truncate text-xs text-slate-500">{item.id}</span>
                        </div>
                      </AccordionTrigger>
                      <AccordionContent>
                        <DraftNodeDiffView nodeID={item.id} lines={item.lines} />
                      </AccordionContent>
                    </AccordionItem>
                  ))}
                </Accordion>
              )}
            </section>
          </div>
        )}

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={pending}>
            Cancel
          </Button>
          {isDraftOutdated ? (
            <div
              className="inline-flex items-center gap-1.5 rounded border border-amber-300 bg-amber-100 px-2 py-1 text-xs text-amber-900"
              title="Current draft is outdated because the live version is newer."
            >
              <TriangleAlert className="h-3.5 w-3.5" />
              Outdated draft
            </div>
          ) : null}
          <LoadingButton
            onClick={handlePublish}
            disabled={!version || !title.trim()}
            loading={pending}
            loadingText="Creating..."
          >
            Create
          </LoadingButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

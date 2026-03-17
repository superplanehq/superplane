import { CanvasesCanvasVersion } from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "@/ui/accordion";
import { TriangleAlert } from "lucide-react";
import { DraftNodeDiffSummary, DraftNodeDiffView } from "./draftNodeDiff";
import { WorkflowMarkdownPreview } from "./WorkflowMarkdownPreview";

interface CreateChangeRequestModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  pending: boolean;
  version?: CanvasesCanvasVersion;
  title: string;
  description: string;
  descriptionMode: "write" | "preview";
  onTitleChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  onDescriptionModeChange: (mode: "write" | "preview") => void;
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
  descriptionMode,
  onTitleChange,
  onDescriptionChange,
  onDescriptionModeChange,
  diffSummary,
  isDraftOutdated = false,
  onPublish,
}: CreateChangeRequestModalProps) {
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
          <DialogDescription>
            Add a title and summary. This will create a change request snapshot from your current draft.
          </DialogDescription>
        </DialogHeader>

        {!version ? (
          <div className="rounded-md border border-slate-200 bg-slate-50 p-3 text-sm text-slate-700">
            Enable edit mode and save your draft before creating a change request.
          </div>
        ) : (
          <div className="space-y-4">
            <div>
              <label className="mb-1 block text-xs font-medium text-slate-700">Title</label>
              <input
                value={title}
                onChange={(event) => onTitleChange(event.target.value)}
                placeholder="Title"
                className="h-10 w-full rounded-md border border-slate-300 px-3 text-sm text-slate-900 focus:border-sky-400 focus:outline-none"
              />
            </div>

            <div>
              <label className="mb-1 block text-xs font-medium text-slate-700">Description</label>
              <Tabs
                value={descriptionMode}
                onValueChange={(value) => onDescriptionModeChange(value as "write" | "preview")}
              >
                <TabsList className="mb-2 h-9 gap-1 rounded-md border border-slate-200 bg-slate-50 p-0.5">
                  <TabsTrigger value="write" className="px-3 text-xs">
                    Write
                  </TabsTrigger>
                  <TabsTrigger value="preview" className="px-3 text-xs">
                    Preview
                  </TabsTrigger>
                </TabsList>
                <TabsContent value="write" className="mt-0">
                  <textarea
                    value={description}
                    onChange={(event) => onDescriptionChange(event.target.value)}
                    rows={8}
                    placeholder="Describe what changed and why."
                    className="w-full rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-900 focus:border-sky-400 focus:outline-none"
                  />
                </TabsContent>
                <TabsContent value="preview" className="mt-0">
                  <div className="min-h-[172px] rounded-md border border-slate-200 bg-slate-50 p-3 text-sm text-slate-900">
                    {description.trim() ? (
                      <WorkflowMarkdownPreview content={description} />
                    ) : (
                      <p className="text-xs text-slate-500">Nothing to preview.</p>
                    )}
                  </div>
                </TabsContent>
              </Tabs>
            </div>

            <section className="rounded-md border border-slate-200 bg-slate-50 p-3">
              <p className="text-xs font-semibold uppercase tracking-wide text-slate-600">Draft Diff Summary</p>
              <div className="mt-2 flex flex-wrap items-center gap-2 text-[11px]">
                <span className="rounded border border-emerald-200 bg-emerald-50 px-2 py-0.5 text-emerald-700">
                  +{diffSummary.addedCount} added
                </span>
                <span className="rounded border border-blue-200 bg-blue-50 px-2 py-0.5 text-blue-700">
                  ~{diffSummary.updatedCount} updated
                </span>
                <span className="rounded border border-red-200 bg-red-50 px-2 py-0.5 text-red-700">
                  -{diffSummary.removedCount} removed
                </span>
              </div>
              {diffSummary.items.length === 0 ? (
                <p className="mt-2 text-xs text-slate-600">No node-level changes detected.</p>
              ) : (
                <Accordion type="multiple" className="mt-2 w-full rounded-md border border-slate-200 px-3">
                  {diffSummary.items.map((item, index) => (
                    <AccordionItem
                      key={`${item.id}-${item.changeType}-${index}`}
                      value={`${item.id}-${item.changeType}-${index}`}
                      className="border-slate-200"
                    >
                      <AccordionTrigger className="py-3 hover:no-underline">
                        <div className="flex items-center gap-2 min-w-0">
                          <span
                            className={`inline-flex min-w-8 justify-center rounded px-1.5 py-0.5 text-[11px] font-semibold ${
                              item.changeType === "removed"
                                ? "bg-amber-100 text-amber-700"
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
          <Button onClick={handlePublish} disabled={!version || pending || !title.trim()}>
            {pending ? "Creating..." : "Create"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

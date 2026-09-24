import { useEffect, useMemo, useState } from "react";

import type { FilesFile } from "@/api-client";
import { Button } from "@/components/ui/button";
import { useWorkOrderFileUpload } from "@/hooks/useWorkOrderFileUpload";
import { workOrderFileDownloadMap } from "@/lib/workOrderFiles";

import { WorkOrderDescription } from "../../WorkOrderDescription";
import { WorkOrderDescriptionEditor } from "../../WorkOrderDescriptionEditor";

const MAX_DESCRIPTION_LENGTH = 5000;

/**
 * Description on the split-run Description tab. Drafts can switch the
 * markdown into the work-order editor without leaving the popup.
 */
export function WorkOrderSplitRunDescription({
  description,
  canEdit = false,
  busy = false,
  collapsible = true,
  onSave,
  files,
  organizationId,
  factoryId,
  orderId,
}: {
  description: string;
  canEdit?: boolean;
  busy?: boolean;
  collapsible?: boolean;
  onSave?: (next: string) => void | Promise<void>;
  files?: FilesFile[];
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
}) {
  const [editing, setEditing] = useState(false);
  const [saved, setSaved] = useState(description);
  const [draft, setDraft] = useState(description);
  const canUpload = Boolean(organizationId && factoryId);
  const fileUrls = useMemo(() => workOrderFileDownloadMap(files), [files]);
  const fileUpload = useWorkOrderFileUpload({
    organizationId: organizationId ?? "",
    factoryId: factoryId ?? "",
    orderId,
  });

  useEffect(() => {
    setSaved(description);
    setDraft(description);
  }, [description]);

  const handleCancel = () => {
    setDraft(saved);
    setEditing(false);
  };

  const handleSave = async () => {
    try {
      await onSave?.(draft);
      setSaved(draft);
      setEditing(false);
    } catch {
      // Caller shows the error. Keep the editor open.
    }
  };

  const body = saved.trim() ? (
    <WorkOrderDescription description={saved} files={files} collapsible={collapsible} />
  ) : (
    <p className="text-[13px] text-muted-foreground">No description yet.</p>
  );

  return (
    <section className="relative" aria-label="Description" data-testid="split-run-description">
      {canEdit && !editing ? (
        <button
          type="button"
          className="absolute top-0 right-0 z-10 text-[13px] leading-6 text-muted-foreground underline underline-offset-2 hover:text-foreground"
          onClick={() => {
            setDraft(saved);
            setEditing(true);
          }}
          data-testid="split-run-description-edit"
        >
          Edit
        </button>
      ) : null}

      {editing ? (
        <div
          className="relative rounded-lg border border-border bg-muted/40 px-3 py-2"
          data-testid="split-run-description-editor"
        >
          <div className="absolute top-2 right-3 z-10 flex items-center gap-1">
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="h-6 px-2 text-[13px] text-muted-foreground hover:bg-transparent hover:text-foreground"
              onClick={handleCancel}
            >
              Cancel
            </Button>
            <Button
              type="button"
              size="sm"
              className="h-6 px-2.5 text-[13px]"
              disabled={busy || fileUpload.isUploading}
              onClick={() => void handleSave()}
            >
              Save
            </Button>
          </div>
          <WorkOrderDescriptionEditor
            value={draft}
            maxLength={MAX_DESCRIPTION_LENGTH}
            disabled={busy || fileUpload.isUploading}
            className="min-h-32 pr-28 text-[13px] leading-[1.625] [&>p:first-child]:mt-0"
            onChange={setDraft}
            fileUrls={fileUrls}
            onUploadFiles={canUpload ? fileUpload.uploadFiles : undefined}
            isUploading={fileUpload.isUploading}
          />
        </div>
      ) : (
        <div className={canEdit ? "pr-10" : undefined}>{body}</div>
      )}
    </section>
  );
}

import type { FactoriesAddWorkOrderArtifactBody, FactoriesWorkOrderArtifactType } from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { LoadingButton } from "@/components/ui/loading-button";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@/components/ui/select";
import { safeExternalUrl } from "@/lib/safeExternalUrl";
import { useState } from "react";

type ArtifactKind = "pr" | "markdown";

interface WorkOrderAttachArtifactDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  isSubmitting: boolean;
  onSubmit: (input: FactoriesAddWorkOrderArtifactBody) => Promise<void>;
}

function artifactKindToProto(kind: ArtifactKind): FactoriesWorkOrderArtifactType {
  return kind === "pr" ? "TYPE_PR" : "TYPE_MARKDOWN";
}

export function WorkOrderAttachArtifactDialog({
  open,
  onOpenChange,
  isSubmitting,
  onSubmit,
}: WorkOrderAttachArtifactDialogProps) {
  const [kind, setKind] = useState<ArtifactKind>("pr");
  const [url, setUrl] = useState("");
  const [title, setTitle] = useState("");
  const [body, setBody] = useState("");

  const resetForm = () => {
    setKind("pr");
    setUrl("");
    setTitle("");
    setBody("");
  };

  const trimmedUrl = url.trim();
  const isUrlSafe = kind === "pr" ? Boolean(safeExternalUrl(trimmedUrl)) : true;
  const showUrlError = kind === "pr" && trimmedUrl !== "" && !isUrlSafe;
  const canSubmit = kind === "pr" ? isUrlSafe && trimmedUrl !== "" : Boolean(body.trim());

  const handleSubmit = async () => {
    if (!canSubmit) {
      return;
    }

    try {
      await onSubmit({
        type: artifactKindToProto(kind),
        url: kind === "pr" ? trimmedUrl : undefined,
        title: title.trim() ? title.trim() : undefined,
        body: kind === "markdown" ? body : undefined,
      });
      resetForm();
      onOpenChange(false);
    } catch {
      // Toast surfaced from the action hook.
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(next) => {
        onOpenChange(next);
        if (!next) {
          resetForm();
        }
      }}
    >
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Attach artifact</DialogTitle>
          <DialogDescription>Capture a PR or a note so the team and the LLM can reference it later.</DialogDescription>
        </DialogHeader>

        <div className="space-y-3">
          <div className="space-y-1.5">
            <label htmlFor="artifact-type" className="text-sm font-medium text-gray-800 dark:text-gray-200">
              Type
            </label>
            <Select value={kind} onValueChange={(value) => setKind(value as ArtifactKind)}>
              <SelectTrigger id="artifact-type" className="h-8 w-full">
                <SelectValue placeholder="Choose type" />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="pr">Pull request</SelectItem>
                <SelectItem value="markdown">Markdown note</SelectItem>
              </SelectContent>
            </Select>
          </div>

          {kind === "pr" ? (
            <div className="space-y-1.5">
              <label htmlFor="artifact-url" className="text-sm font-medium text-gray-800 dark:text-gray-200">
                URL <span className="text-red-500">*</span>
              </label>
              <Input
                id="artifact-url"
                type="url"
                value={url}
                onChange={(event) => setUrl(event.target.value)}
                placeholder="https://github.com/org/repo/pull/1"
                disabled={isSubmitting}
                aria-invalid={showUrlError || undefined}
                aria-describedby={showUrlError ? "artifact-url-error" : undefined}
              />
              {showUrlError ? (
                <p id="artifact-url-error" className="text-xs text-red-600 dark:text-red-400">
                  Enter a full http:// or https:// link.
                </p>
              ) : null}
            </div>
          ) : null}

          <div className="space-y-1.5">
            <label htmlFor="artifact-title" className="text-sm font-medium text-gray-800 dark:text-gray-200">
              Title
            </label>
            <Input
              id="artifact-title"
              value={title}
              onChange={(event) => setTitle(event.target.value)}
              placeholder={kind === "pr" ? "Add stripe checkout endpoint" : "Design notes"}
              disabled={isSubmitting}
            />
          </div>

          {kind === "markdown" ? (
            <div className="space-y-1.5">
              <label htmlFor="artifact-body" className="text-sm font-medium text-gray-800 dark:text-gray-200">
                Body <span className="text-red-500">*</span>
              </label>
              <textarea
                id="artifact-body"
                value={body}
                onChange={(event) => setBody(event.target.value)}
                rows={5}
                placeholder="Write markdown..."
                className="block w-full resize-y rounded-md border border-gray-300 bg-white px-3 py-2 text-sm text-gray-900 focus:border-gray-500 focus:outline-none dark:border-gray-600/70 dark:bg-gray-800 dark:text-gray-100"
                disabled={isSubmitting}
              />
            </div>
          ) : null}
        </div>

        <DialogFooter>
          <DialogClose asChild>
            <Button type="button" variant="ghost">
              Cancel
            </Button>
          </DialogClose>
          <LoadingButton
            type="button"
            disabled={!canSubmit || isSubmitting}
            loading={isSubmitting}
            loadingText="Attaching..."
            onClick={handleSubmit}
            data-testid="work-order-attach-artifact-submit"
          >
            Attach
          </LoadingButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

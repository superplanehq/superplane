import { NodeViewWrapper, ReactNodeViewRenderer, type NodeViewProps } from "@tiptap/react";
import { Trash2 } from "lucide-react";

import { parseWorkOrderFileId, resolveWorkOrderFileSrc, isWorkOrderVideoSource } from "@/lib/workOrderFiles";
import { WorkOrderVideo } from "@/pages/app/WorkOrderVideo";

import { CREATE_WORK_ORDER_REQUEST_COPY } from "../createWorkOrderRequestCopy";
import { WorkOrderImage } from "./workOrderDescriptionImage";

function WorkOrderRequestImageView({ node, deleteNode, editor }: NodeViewProps) {
  const rawSrc = node.attrs.src as string | undefined;
  const src =
    (node.attrs.resolvedSrc as string | null | undefined) ??
    resolveWorkOrderFileSrc(rawSrc, editor.storage.image?.downloadUrls);
  const alt = (node.attrs.alt as string | undefined) ?? "";
  const id = parseWorkOrderFileId(rawSrc) ?? rawSrc ?? "image";

  const contentType = id ? editor.storage.image?.contentTypes?.[id] : undefined;
  const media = isWorkOrderVideoSource({ contentType, src, alt }) ? (
    <WorkOrderVideo src={src} className="work-order-file-image" alt={alt} contentType={contentType} />
  ) : (
    <img src={src} alt={alt} className="work-order-file-image" />
  );

  return (
    <NodeViewWrapper className="work-order-request-image group relative inline-block max-w-full">
      {media}
      {editor.isEditable ? (
        <button
          type="button"
          className="absolute top-1.5 right-1.5 flex size-6 items-center justify-center rounded-full bg-foreground/75 text-background opacity-0 transition-opacity group-hover:opacity-100 focus-visible:opacity-100"
          aria-label={CREATE_WORK_ORDER_REQUEST_COPY.deleteImage}
          data-testid={`create-work-order-request-inline-image-remove-${id}`}
          onClick={(event) => {
            event.preventDefault();
            event.stopPropagation();
            deleteNode();
          }}
        >
          <Trash2 className="size-3.5" aria-hidden />
        </button>
      ) : null}
    </NodeViewWrapper>
  );
}

export const WorkOrderRequestImage = WorkOrderImage.extend({
  addNodeView() {
    return ReactNodeViewRenderer(WorkOrderRequestImageView);
  },
});

import { mergeAttributes } from "@tiptap/core";
import Image from "@tiptap/extension-image";
import { NodeViewWrapper, ReactNodeViewRenderer, type NodeViewProps } from "@tiptap/react";

import {
  isBrowserPlayableWorkOrderVideo,
  isWorkOrderVideoSource,
  parseWorkOrderFileId,
  resolveWorkOrderFileSrc,
} from "@/lib/workOrderFiles";
import { WorkOrderVideo } from "@/pages/app/WorkOrderVideo";

declare module "@tiptap/core" {
  interface Storage {
    image: {
      downloadUrls: Record<string, string>;
      contentTypes: Record<string, string>;
    };
  }
}

function WorkOrderMediaView({ node, editor }: NodeViewProps) {
  const rawSrc = node.attrs.src as string | undefined;
  const src =
    (node.attrs.resolvedSrc as string | null | undefined) ??
    resolveWorkOrderFileSrc(rawSrc, editor.storage.image?.downloadUrls);
  const alt = (node.attrs.alt as string | undefined) ?? "";
  const id = parseWorkOrderFileId(rawSrc);
  const contentType = id ? editor.storage.image?.contentTypes?.[id] : undefined;
  const media = isWorkOrderVideoSource({ contentType, src, alt }) ? (
    <WorkOrderVideo src={src} className="work-order-file-image" alt={alt} contentType={contentType} />
  ) : (
    <img src={src} alt={alt} className="work-order-file-image" />
  );
  return <NodeViewWrapper className="inline-block max-w-full">{media}</NodeViewWrapper>;
}

export const WorkOrderImage = Image.extend({
  addStorage() {
    return {
      downloadUrls: {} as Record<string, string>,
      contentTypes: {} as Record<string, string>,
    };
  },
  addAttributes() {
    return {
      ...this.parent?.(),
      resolvedSrc: {
        default: null,
        rendered: false,
      },
    };
  },
  addNodeView() {
    return ReactNodeViewRenderer(WorkOrderMediaView);
  },
  renderHTML({ HTMLAttributes }) {
    const { resolvedSrc, ...rest } = HTMLAttributes;
    const src =
      (resolvedSrc as string | undefined | null) ??
      resolveWorkOrderFileSrc(rest.src as string | undefined, this.editor?.storage.image?.downloadUrls);
    const fileId = parseWorkOrderFileId(rest.src as string | undefined);
    const contentType = fileId ? this.editor?.storage.image?.contentTypes?.[fileId] : undefined;
    const attrs = mergeAttributes(this.options.HTMLAttributes, rest, {
      src,
      class: "work-order-file-image",
    });
    if (isWorkOrderVideoSource({ contentType, src, alt: rest.alt as string | undefined })) {
      if (isBrowserPlayableWorkOrderVideo(contentType, src, rest.alt as string | undefined)) {
        return ["video", mergeAttributes(attrs, { controls: "", playsinline: "" })];
      }
      return ["span", { class: "work-order-video-fallback" }, rest.alt || "Video"];
    }
    return ["img", attrs];
  },
}).configure({
  inline: false,
  allowBase64: false,
});

import { mergeAttributes } from "@tiptap/core";
import Image from "@tiptap/extension-image";
import { NodeViewWrapper, ReactNodeViewRenderer, type NodeViewProps } from "@tiptap/react";

import {
  isBrowserPlayableWorkOrderAudio,
  isBrowserPlayableWorkOrderVideo,
  isWorkOrderAudioSource,
  isWorkOrderMediaSource,
  parseWorkOrderFileId,
  resolveWorkOrderFileSrc,
} from "@/lib/workOrderFiles";
import { HOSTED_VIDEO_COPY, parseHostedVideoUrl } from "@/lib/hostedVideo";
import { WorkOrderVideo } from "@/pages/app/WorkOrderVideo";
import { HostedVideoEmbed } from "@/pages/app/HostedVideoEmbed";

declare module "@tiptap/core" {
  interface Storage {
    image: {
      downloadUrls: Record<string, string>;
      contentTypes: Record<string, string>;
    };
  }
}

function workOrderMediaHTML(
  attrs: Record<string, unknown>,
  contentType: string | undefined,
  src: string | undefined,
  alt: string | undefined,
): readonly [string, ...unknown[]] {
  if (isWorkOrderAudioSource({ contentType, src, alt })) {
    if (isBrowserPlayableWorkOrderAudio(contentType, src, alt)) {
      return ["audio", mergeAttributes(attrs, { controls: "" })];
    }
    return ["span", { class: "work-order-video-fallback" }, alt || "Audio"];
  }
  if (isBrowserPlayableWorkOrderVideo(contentType, src, alt)) {
    return ["video", mergeAttributes(attrs, { controls: "", playsinline: "" })];
  }
  return ["span", { class: "work-order-video-fallback" }, alt || "Video"];
}

function WorkOrderMediaView({ node, editor }: NodeViewProps) {
  const rawSrc = node.attrs.src as string | undefined;
  const hosted = parseHostedVideoUrl(rawSrc ?? "");
  if (hosted) {
    return (
      <NodeViewWrapper className="inline-block max-w-full">
        <HostedVideoEmbed video={hosted} className="work-order-file-image" />
      </NodeViewWrapper>
    );
  }
  const src =
    (node.attrs.resolvedSrc as string | null | undefined) ??
    resolveWorkOrderFileSrc(rawSrc, editor.storage.image?.downloadUrls);
  const alt = (node.attrs.alt as string | undefined) ?? "";
  const id = parseWorkOrderFileId(rawSrc);
  const contentType = id ? editor.storage.image?.contentTypes?.[id] : undefined;
  const media = isWorkOrderMediaSource({ contentType, src, alt }) ? (
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
    const hosted = parseHostedVideoUrl(String(rest.src ?? ""));
    if (hosted?.embedUrl) {
      return [
        "iframe",
        {
          src: hosted.embedUrl,
          title: hosted.providerName,
          class: "hosted-video-embed aspect-video w-full max-w-xl rounded-md border border-border",
          allowfullscreen: "true",
        },
      ];
    }
    if (hosted) {
      return [
        "span",
        { class: "hosted-video-card" },
        ["span", {}, hosted.providerName],
        ["a", { href: hosted.pageUrl, target: "_blank", rel: "noopener noreferrer" }, HOSTED_VIDEO_COPY.open],
      ];
    }
    const src =
      (resolvedSrc as string | undefined | null) ??
      resolveWorkOrderFileSrc(rest.src as string | undefined, this.editor?.storage.image?.downloadUrls);
    const fileId = parseWorkOrderFileId(rest.src as string | undefined);
    const contentType = fileId ? this.editor?.storage.image?.contentTypes?.[fileId] : undefined;
    const attrs = mergeAttributes(this.options.HTMLAttributes, rest, {
      src,
      class: "work-order-file-image",
    });
    if (isWorkOrderMediaSource({ contentType, src, alt: rest.alt as string | undefined })) {
      return workOrderMediaHTML(attrs, contentType, src, rest.alt as string | undefined);
    }
    return ["img", attrs];
  },
}).configure({
  inline: false,
  allowBase64: false,
});

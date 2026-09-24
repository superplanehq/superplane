import { mergeAttributes } from "@tiptap/core";
import Image from "@tiptap/extension-image";

import { resolveWorkOrderFileSrc } from "@/lib/workOrderFiles";

declare module "@tiptap/core" {
  interface Storage {
    image: {
      downloadUrls: Record<string, string>;
    };
  }
}

export const WorkOrderImage = Image.extend({
  addStorage() {
    return {
      downloadUrls: {} as Record<string, string>,
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
  renderHTML({ HTMLAttributes }) {
    const { resolvedSrc, ...rest } = HTMLAttributes;
    const src =
      (resolvedSrc as string | undefined | null) ??
      resolveWorkOrderFileSrc(rest.src as string | undefined, this.editor?.storage.image?.downloadUrls);
    return [
      "img",
      mergeAttributes(this.options.HTMLAttributes, rest, {
        src,
        class: "work-order-file-image",
      }),
    ];
  },
}).configure({
  inline: false,
  allowBase64: false,
});

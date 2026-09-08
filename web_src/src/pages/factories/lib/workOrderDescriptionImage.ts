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
  renderHTML({ HTMLAttributes }) {
    const src = resolveWorkOrderFileSrc(
      HTMLAttributes.src as string | undefined,
      this.editor?.storage.image?.downloadUrls,
    );
    return [
      "img",
      mergeAttributes(this.options.HTMLAttributes, HTMLAttributes, {
        src,
        class: "work-order-file-image",
      }),
    ];
  },
}).configure({
  inline: false,
  allowBase64: false,
});

import type { Editor } from "@tiptap/core";
import Image from "@tiptap/extension-image";
import { Plugin, PluginKey } from "@tiptap/pm/state";
import { Decoration, DecorationSet } from "@tiptap/pm/view";

import { resolveWorkOrderFileSrc } from "@/lib/workOrderFiles";

/**
 * `sp-file://` refs are stable identifiers that get persisted in the
 * markdown. The actual (short-lived, signed) download URL for a file is only
 * known once the work order's files have loaded, so the editor resolves the
 * displayed `src` through a ProseMirror decoration instead of baking it into
 * the node itself. Decorations are recomputed whenever the download URL map
 * changes, which keeps images from looking "broken" while editing even
 * though the underlying node still stores the original `sp-file://` ref.
 */
export const workOrderImageDownloadUrlsPluginKey = new PluginKey<Record<string, string>>("workOrderImageDownloadUrls");

export const WorkOrderImage = Image.extend({
  addProseMirrorPlugins() {
    return [
      ...(this.parent?.() ?? []),
      new Plugin({
        key: workOrderImageDownloadUrlsPluginKey,
        state: {
          init: () => ({}) as Record<string, string>,
          apply(tr, value) {
            const meta = tr.getMeta(workOrderImageDownloadUrlsPluginKey) as Record<string, string> | undefined;
            return meta ?? value;
          },
        },
        props: {
          decorations(state) {
            const downloadUrls = workOrderImageDownloadUrlsPluginKey.getState(state);
            if (!downloadUrls || Object.keys(downloadUrls).length === 0) {
              return null;
            }
            const decorations: Decoration[] = [];
            state.doc.descendants((node, pos) => {
              if (node.type.name !== "image") {
                return;
              }
              const src = node.attrs.src as string | undefined;
              const resolved = resolveWorkOrderFileSrc(src, downloadUrls);
              if (resolved && resolved !== src) {
                decorations.push(Decoration.node(pos, pos + node.nodeSize, { src: resolved }));
              }
            });
            return decorations.length ? DecorationSet.create(state.doc, decorations) : null;
          },
        },
      }),
    ];
  },
}).configure({
  inline: false,
  allowBase64: false,
  HTMLAttributes: {
    class: "work-order-file-image",
  },
});

/**
 * Pushes the current file id -> download URL map into the editor so the
 * `WorkOrderImage` decoration plugin can resolve `sp-file://` refs to real
 * URLs. Safe to call before the map is fully populated (e.g. while files are
 * still loading) since it simply re-runs whenever the map changes.
 */
export function setWorkOrderImageDownloadUrls(editor: Editor, downloadUrls: Record<string, string>): void {
  if (editor.isDestroyed) {
    return;
  }
  const tr = editor.state.tr.setMeta(workOrderImageDownloadUrlsPluginKey, downloadUrls);
  tr.setMeta("addToHistory", false);
  editor.view.dispatch(tr);
}

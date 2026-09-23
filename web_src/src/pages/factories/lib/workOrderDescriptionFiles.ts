import type { Editor } from "@tiptap/core";

import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";

export function insertUploadedFiles(editor: Editor, files: UploadedWorkOrderFile[]): void {
  const contentTypes = { ...(editor.storage.image?.contentTypes ?? {}) };
  for (const file of files) {
    contentTypes[file.id] = file.contentType;
  }
  editor.storage.image = {
    ...(editor.storage.image ?? { downloadUrls: {} }),
    contentTypes,
  };
  for (const file of files) {
    if (file.isImage || file.isVideo) {
      editor.chain().focus().setImage({ src: file.ref, alt: file.filename }).run();
    } else {
      editor.chain().focus().insertContent(`[${file.filename}](${file.ref})`).run();
    }
  }
}

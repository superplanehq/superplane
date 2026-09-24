import type { Editor } from "@tiptap/core";

import type { UploadedWorkOrderFile } from "@/hooks/useWorkOrderFileUpload";

export function insertUploadedFiles(editor: Editor, files: UploadedWorkOrderFile[]): void {
  for (const file of files) {
    if (file.isImage) {
      editor.chain().focus().setImage({ src: file.ref, alt: file.filename }).run();
    } else {
      editor.chain().focus().insertContent(`[${file.filename}](${file.ref})`).run();
    }
  }
}

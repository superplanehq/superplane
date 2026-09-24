import { Editor } from "@tiptap/core";
import StarterKit from "@tiptap/starter-kit";
import { describe, expect, it } from "bun:test";
import { insertUploadedFiles } from "./workOrderDescriptionFiles";
import { WorkOrderImage } from "./workOrderDescriptionImage";

function markdownEditor() {
  return new Editor({
    extensions: [StarterKit, WorkOrderImage],
    content: "",
  });
}

describe("insertUploadedFiles", () => {
  it("inserts an image markdown node for image uploads", () => {
    const editor = markdownEditor();
    insertUploadedFiles(editor, [
      {
        id: "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
        filename: "shot.png",
        contentType: "image/png",
        ref: "sp-file://aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
        previewUrl: "blob:preview",
        isImage: true,
      },
    ]);

    expect(editor.getHTML()).toContain('src="sp-file://aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"');
    editor.destroy();
  });

  it("inserts a markdown link for non-image uploads", () => {
    const editor = markdownEditor();
    insertUploadedFiles(editor, [
      {
        id: "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
        filename: "notes.txt",
        contentType: "text/plain",
        ref: "sp-file://aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
        previewUrl: "blob:preview",
        isImage: false,
      },
    ]);

    expect(editor.getHTML()).toContain("notes.txt");
    expect(editor.getHTML()).toContain("sp-file://aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee");
    editor.destroy();
  });
});

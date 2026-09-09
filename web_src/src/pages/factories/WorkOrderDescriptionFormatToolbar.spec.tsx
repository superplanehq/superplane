import { Editor } from "@tiptap/core";
import StarterKit from "@tiptap/starter-kit";
import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "vitest";

import { WorkOrderDescriptionFormatToolbar } from "./WorkOrderDescriptionFormatToolbar";
import { WorkOrderImage } from "./lib/workOrderDescriptionImage";

function markdownEditor() {
  return new Editor({
    extensions: [StarterKit, WorkOrderImage],
    content: "Hello world",
  });
}

afterEach(() => {
  cleanup();
});

describe("WorkOrderDescriptionFormatToolbar attach control", () => {
  it("does not render an attach button when no upload handler is given", () => {
    const editor = markdownEditor();

    render(<WorkOrderDescriptionFormatToolbar editor={editor} disabled={false} />);

    expect(screen.queryByRole("button", { name: "Attach file" })).not.toBeInTheDocument();
    editor.destroy();
  });

  it("uploads a selected image and inserts it as an sp-file ref", async () => {
    const editor = markdownEditor();
    const onUploadFiles = vi.fn().mockResolvedValue([
      {
        id: "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
        filename: "shot.png",
        contentType: "image/png",
        ref: "sp-file://aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
        previewUrl: "blob:preview",
        isImage: true,
      },
    ]);

    render(<WorkOrderDescriptionFormatToolbar editor={editor} disabled={false} onUploadFiles={onUploadFiles} />);

    fireEvent.click(screen.getByRole("button", { name: "Attach file" }));

    const input = screen.getByTestId("work-order-description-file-input");
    const file = new File(["png"], "shot.png", { type: "image/png" });
    fireEvent.change(input, { target: { files: [file] } });

    expect(onUploadFiles).toHaveBeenCalledTimes(1);
    const uploaded = await onUploadFiles.mock.results[0].value;
    expect(uploaded).toHaveLength(1);

    await vi.waitFor(() => {
      expect(editor.getHTML()).toContain('src="sp-file://aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee"');
    });
    editor.destroy();
  });

  it("uploads a selected text file and inserts it as a markdown link", async () => {
    const editor = markdownEditor();
    const onUploadFiles = vi.fn().mockResolvedValue([
      {
        id: "aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
        filename: "notes.txt",
        contentType: "text/plain",
        ref: "sp-file://aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee",
        previewUrl: "blob:preview",
        isImage: false,
      },
    ]);

    render(<WorkOrderDescriptionFormatToolbar editor={editor} disabled={false} onUploadFiles={onUploadFiles} />);

    fireEvent.click(screen.getByRole("button", { name: "Attach file" }));
    const input = screen.getByTestId("work-order-description-file-input");
    const file = new File(["notes"], "notes.txt", { type: "text/plain" });
    fireEvent.change(input, { target: { files: [file] } });

    await vi.waitFor(() => {
      expect(editor.getHTML()).toContain("sp-file://aaaaaaaa-bbbb-4ccc-8ddd-eeeeeeeeeeee");
    });
    expect(editor.getHTML()).toContain("notes.txt");
    editor.destroy();
  });

  it("disables the attach button while disabled", () => {
    const editor = markdownEditor();

    render(<WorkOrderDescriptionFormatToolbar editor={editor} disabled={true} onUploadFiles={vi.fn()} />);

    expect(screen.getByRole("button", { name: "Attach file" })).toBeDisabled();
    editor.destroy();
  });
});

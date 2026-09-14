import { describe, expect, it } from "bun:test";

import { appendUploadedWorkOrderFiles } from "./createWorkOrderRequestFiles";

describe("appendUploadedWorkOrderFiles", () => {
  it("appends an image markdown block after the message", () => {
    expect(
      appendUploadedWorkOrderFiles("Refunds fail on retry.", [
        {
          id: "file-1",
          filename: "checkout.png",
          contentType: "image/png",
          ref: "sp-file://file-1",
          previewUrl: "https://cdn.example.com/checkout.png",
          isImage: true,
        },
      ]),
    ).toBe("Refunds fail on retry.\n\n![checkout.png](sp-file://file-1)");
  });

  it("appends a file link when the upload is not an image", () => {
    expect(
      appendUploadedWorkOrderFiles("", [
        {
          id: "file-2",
          filename: "notes.md",
          contentType: "text/markdown",
          ref: "sp-file://file-2",
          previewUrl: "https://cdn.example.com/notes.md",
          isImage: false,
        },
      ]),
    ).toBe("[notes.md](sp-file://file-2)");
  });
});

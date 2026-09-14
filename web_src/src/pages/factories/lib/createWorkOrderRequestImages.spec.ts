import { describe, expect, it } from "bun:test";

import { MAX_IMAGE_ATTACHMENTS } from "@/components/AgentSidebar/useImageAttachments";

import {
  appendUploadedWorkOrderImages,
  countCreateWorkOrderRequestImages,
  createWorkOrderRequestImages,
  mergeCreateWorkOrderRequestImages,
  removeCreateWorkOrderRequestMarkdownImage,
  selectCreateWorkOrderRequestUploads,
} from "./createWorkOrderRequestImages";

describe("createWorkOrderRequestImages", () => {
  it("collects unique markdown images with resolved file URLs", () => {
    expect(
      createWorkOrderRequestImages(
        "Refunds fail.\n\n![Checkout](sp-file://file-1)\n\n![Checkout](sp-file://file-1)\n\n![Receipt](sp-file://file-2)",
        {
          "file-1": "https://cdn.example.com/checkout.png",
          "file-2": "https://cdn.example.com/receipt.png",
        },
      ),
    ).toEqual([
      { id: "file-1", alt: "Checkout", src: "https://cdn.example.com/checkout.png" },
      { id: "file-2", alt: "Receipt", src: "https://cdn.example.com/receipt.png" },
    ]);
  });

  it("skips markdown that has no images", () => {
    expect(createWorkOrderRequestImages("Refunds fail on retry.")).toEqual([]);
  });
});

describe("mergeCreateWorkOrderRequestImages", () => {
  it("adds attach-only images after description images and skips duplicates", () => {
    expect(
      mergeCreateWorkOrderRequestImages(
        [{ id: "file-1", alt: "Checkout", src: "https://cdn.example.com/checkout.png" }],
        [
          {
            id: "file-1",
            filename: "checkout.png",
            contentType: "image/png",
            ref: "sp-file://file-1",
            previewUrl: "https://cdn.example.com/checkout.png",
            isImage: true,
          },
          {
            id: "file-2",
            filename: "receipt.png",
            contentType: "image/png",
            ref: "sp-file://file-2",
            previewUrl: "https://cdn.example.com/receipt.png",
            isImage: true,
          },
          {
            id: "file-3",
            filename: "notes.md",
            contentType: "text/markdown",
            ref: "sp-file://file-3",
            previewUrl: "https://cdn.example.com/notes.md",
            isImage: false,
          },
        ],
      ),
    ).toEqual([
      { id: "file-1", alt: "Checkout", src: "https://cdn.example.com/checkout.png" },
      { id: "file-2", alt: "receipt.png", src: "https://cdn.example.com/receipt.png" },
    ]);
  });
});

describe("appendUploadedWorkOrderImages", () => {
  it("appends image markdown and strips label delimiters", () => {
    expect(
      appendUploadedWorkOrderImages("Refunds fail on retry.", [
        {
          id: "file-1",
          filename: "check]out(1).png",
          contentType: "image/png",
          ref: "sp-file://file-1",
          previewUrl: "https://cdn.example.com/checkout.png",
          isImage: true,
        },
        {
          id: "file-2",
          filename: "notes.md",
          contentType: "text/markdown",
          ref: "sp-file://file-2",
          previewUrl: "https://cdn.example.com/notes.md",
          isImage: false,
        },
      ]),
    ).toBe("Refunds fail on retry.\n\n![checkout1.png](sp-file://file-1)");
  });
});

describe("countCreateWorkOrderRequestImages", () => {
  it("counts unique markdown images even when the preview URL is missing", () => {
    expect(countCreateWorkOrderRequestImages("![Checkout](sp-file://file-1)\n\n![Receipt](sp-file://file-2)")).toBe(2);
  });

  it("does not double-count an attach-only file that is already in the description", () => {
    expect(
      countCreateWorkOrderRequestImages("![Checkout](sp-file://file-1)", [
        {
          id: "file-1",
          filename: "checkout.png",
          contentType: "image/png",
          ref: "sp-file://file-1",
          previewUrl: "https://cdn.example.com/checkout.png",
          isImage: true,
        },
        {
          id: "file-2",
          filename: "receipt.png",
          contentType: "image/png",
          ref: "sp-file://file-2",
          previewUrl: "https://cdn.example.com/receipt.png",
          isImage: true,
        },
      ]),
    ).toBe(2);
  });
});

describe("selectCreateWorkOrderRequestUploads", () => {
  it("keeps only the remaining image slots before upload", () => {
    const files = Array.from(
      { length: 3 },
      (_, index) => new File(["img"], `shot-${index}.png`, { type: "image/png" }),
    );

    const selected = selectCreateWorkOrderRequestUploads(files, MAX_IMAGE_ATTACHMENTS - 1);

    expect(selected.accepted).toEqual([files[0]]);
    expect(selected.rejectedCount).toBe(2);
  });

  it("rejects every image when the stack is already full", () => {
    const files = [new File(["img"], "shot.png", { type: "image/png" })];

    expect(selectCreateWorkOrderRequestUploads(files, MAX_IMAGE_ATTACHMENTS)).toEqual({
      accepted: [],
      rejectedCount: 1,
    });
  });

  it("still accepts non-image files when the image stack is full", () => {
    const note = new File(["notes"], "notes.md", { type: "text/markdown" });
    const shot = new File(["img"], "shot.png", { type: "image/png" });

    expect(selectCreateWorkOrderRequestUploads([shot, note], MAX_IMAGE_ATTACHMENTS)).toEqual({
      accepted: [note],
      rejectedCount: 1,
    });
  });
});

describe("removeCreateWorkOrderRequestMarkdownImage", () => {
  it("removes the matching image block and keeps the surrounding text", () => {
    expect(
      removeCreateWorkOrderRequestMarkdownImage(
        "Refunds fail.\n\n![Checkout](sp-file://file-1)\n\nMore context.",
        "file-1",
      ),
    ).toBe("Refunds fail.\n\nMore context.");
  });
});

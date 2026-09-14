import { describe, expect, it } from "bun:test";

import {
  createWorkOrderRequestImages,
  mergeCreateWorkOrderRequestImages,
  removeCreateWorkOrderRequestMarkdownImage,
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

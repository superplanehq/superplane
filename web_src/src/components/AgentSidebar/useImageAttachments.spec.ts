import { act, renderHook } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { MAX_IMAGE_ATTACHMENTS, MAX_TOTAL_IMAGE_BYTES, useImageAttachments } from "./useImageAttachments";

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
}));

describe("useImageAttachments", () => {
  beforeEach(() => {
    vi.restoreAllMocks();
  });

  it("enforces the image count limit across overlapping file additions", async () => {
    const { result } = renderHook(() => useImageAttachments());

    await act(async () => {
      await Promise.all([result.current.addFiles(makeImages(5, 1)), result.current.addFiles(makeImages(5, 1))]);
    });

    expect(result.current.images).toHaveLength(MAX_IMAGE_ATTACHMENTS);
  });

  it("enforces the total byte limit across overlapping file additions", async () => {
    const { result } = renderHook(() => useImageAttachments());
    const imageBytes = Math.floor(MAX_TOTAL_IMAGE_BYTES / 2) + 1;

    await act(async () => {
      await Promise.all([
        result.current.addFiles(makeImages(1, imageBytes)),
        result.current.addFiles(makeImages(1, imageBytes)),
      ]);
    });

    expect(result.current.images).toHaveLength(1);
    expect(totalBytes(result.current.images)).toBeLessThanOrEqual(MAX_TOTAL_IMAGE_BYTES);
  });
});

function makeImages(count: number, bytes: number): File[] {
  return Array.from(
    { length: count },
    (_, index) => new File([new Uint8Array(bytes)], `image-${index}.png`, { type: "image/png" }),
  );
}

function totalBytes(images: Array<{ bytes: number }>): number {
  return images.reduce((sum, image) => sum + image.bytes, 0);
}

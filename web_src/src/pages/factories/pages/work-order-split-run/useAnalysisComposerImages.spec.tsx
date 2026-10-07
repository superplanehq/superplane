import { act, renderHook } from "@testing-library/react";
import { describe, expect, it, vi } from "bun:test";

import { useAnalysisComposerImages } from "./useAnalysisComposerImages";

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
}));

function hostedComposer(count: number): string {
  return Array.from({ length: count }, (_, index) => {
    const id = `abcdefghij${index}`;
    return `![YouTube](https://www.youtube.com/watch?v=${id})`;
  }).join("\n\n");
}

describe("useAnalysisComposerImages", () => {
  it("counts composer hosted videos when choosing uploads", async () => {
    const onUploadFiles = vi.fn(async (files: File[]) =>
      files.map((file, index) => ({
        id: `file-${index}`,
        filename: file.name,
        contentType: file.type,
        ref: `sp-file://file-${index}`,
        previewUrl: "blob:preview",
        isImage: true,
      })),
    );
    const { result } = renderHook(() =>
      useAnalysisComposerImages({
        disabled: false,
        onUploadFiles,
        markdown: hostedComposer(7),
      }),
    );
    const files = [new File(["a"], "a.png", { type: "image/png" }), new File(["b"], "b.png", { type: "image/png" })];

    await act(async () => {
      await result.current.attach(files);
    });

    expect(onUploadFiles).toHaveBeenCalledOnce();
    expect(onUploadFiles.mock.calls[0]?.[0]).toHaveLength(1);
  });

  it("rejects visual uploads when composer links already fill the limit", async () => {
    const onUploadFiles = vi.fn();
    const { result } = renderHook(() =>
      useAnalysisComposerImages({
        disabled: false,
        onUploadFiles,
        markdown: hostedComposer(8),
      }),
    );

    await act(async () => {
      await result.current.attach([new File(["img"], "shot.png", { type: "image/png" })]);
    });

    expect(onUploadFiles).not.toHaveBeenCalled();
  });
});

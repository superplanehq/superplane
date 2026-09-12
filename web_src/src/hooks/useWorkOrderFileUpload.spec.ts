import { act, renderHook, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

const { filesCreateFactoryFile, filesCreateWorkOrderFile, showErrorToast } = vi.hoisted(() => ({
  filesCreateFactoryFile: vi.fn(),
  filesCreateWorkOrderFile: vi.fn(),
  showErrorToast: vi.fn(),
}));

vi.mock("@/api-client", () => ({
  filesCreateFactoryFile,
  filesCreateWorkOrderFile,
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast,
}));

import { useWorkOrderFileUpload } from "./useWorkOrderFileUpload";

describe("useWorkOrderFileUpload", () => {
  beforeEach(() => {
    filesCreateFactoryFile.mockReset();
    filesCreateWorkOrderFile.mockReset();
    showErrorToast.mockReset();
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue({
        ok: true,
      }),
    );
    vi.spyOn(URL, "createObjectURL").mockReturnValue("blob:preview");
  });

  it("uploads a workspace file and returns an sp-file ref", async () => {
    filesCreateFactoryFile.mockResolvedValue({ data: { file: { id: "file-1" } } });
    const { result } = renderHook(() => useWorkOrderFileUpload({ organizationId: "org-1", factoryId: "factory-1" }));

    let uploaded: Awaited<ReturnType<typeof result.current.uploadFiles>> = [];
    await act(async () => {
      uploaded = await result.current.uploadFiles([new File(["png"], "bug.png", { type: "image/png" })]);
    });

    expect(filesCreateFactoryFile).toHaveBeenCalled();
    expect(filesCreateWorkOrderFile).not.toHaveBeenCalled();
    expect(uploaded).toEqual([
      expect.objectContaining({
        id: "file-1",
        ref: "sp-file://file-1",
        isImage: true,
      }),
    ]);
    await waitFor(() => expect(fetch).toHaveBeenCalledWith("/api/v1/files/file-1/content", expect.any(Object)));
  });

  it("rejects a file type that SuperPlane does not store", async () => {
    const { result } = renderHook(() => useWorkOrderFileUpload({ organizationId: "org-1", factoryId: "factory-1" }));

    let uploaded: Awaited<ReturnType<typeof result.current.uploadFiles>> = [];
    await act(async () => {
      uploaded = await result.current.uploadFiles([new File(["x"], "clip.mp4", { type: "video/mp4" })]);
    });

    expect(uploaded).toEqual([]);
    expect(showErrorToast).toHaveBeenCalledWith("This file type is not allowed.");
    expect(filesCreateFactoryFile).not.toHaveBeenCalled();
  });
});

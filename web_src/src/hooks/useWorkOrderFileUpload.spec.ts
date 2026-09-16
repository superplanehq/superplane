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
    const id = "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa";
    const uploadUrl = `https://files.example/api/v1/files/${id}/content`;
    filesCreateFactoryFile.mockResolvedValue({ data: { file: { id, uploadUrl } } });
    const { result } = renderHook(() => useWorkOrderFileUpload({ organizationId: "org-1", factoryId: "factory-1" }));

    let uploaded: Awaited<ReturnType<typeof result.current.uploadFiles>> = [];
    await act(async () => {
      uploaded = await result.current.uploadFiles([new File(["png"], "bug.png", { type: "image/png" })]);
    });

    expect(filesCreateFactoryFile).toHaveBeenCalled();
    expect(filesCreateWorkOrderFile).not.toHaveBeenCalled();
    expect(uploaded).toEqual([
      expect.objectContaining({
        id,
        ref: `sp-file://${id}`,
        isImage: true,
      }),
    ]);
    await waitFor(() => expect(fetch).toHaveBeenCalledWith(uploadUrl, expect.any(Object)));
  });

  it("does not upload content when the API omits the minted upload URL", async () => {
    filesCreateFactoryFile.mockResolvedValue({
      data: { file: { id: "aaaaaaaa-aaaa-4aaa-8aaa-aaaaaaaaaaaa" } },
    });
    const { result } = renderHook(() => useWorkOrderFileUpload({ organizationId: "org-1", factoryId: "factory-1" }));

    let uploaded: Awaited<ReturnType<typeof result.current.uploadFiles>> = [];
    await act(async () => {
      uploaded = await result.current.uploadFiles([new File(["png"], "bug.png", { type: "image/png" })]);
    });

    expect(uploaded).toEqual([]);
    expect(fetch).not.toHaveBeenCalled();
    expect(showErrorToast).toHaveBeenCalledWith("The file could not be stored.");
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

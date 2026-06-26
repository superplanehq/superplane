import { describe, expect, it, vi } from "vitest";

const { toastMock } = vi.hoisted(() => ({
  toastMock: {
    error: vi.fn(),
    success: vi.fn(),
    info: vi.fn(),
  },
}));

vi.mock("sonner", () => ({
  toast: toastMock,
}));

import { showErrorToast, showInfoToast, showSuccessToast } from "@/lib/toast";

describe("toast", () => {
  it("delegates each toast helper to sonner", () => {
    showErrorToast("error");
    showSuccessToast("success", { id: "status-id" });
    showInfoToast("info", { id: "info-id" });

    expect(toastMock.error).toHaveBeenCalledWith("error", undefined);
    expect(toastMock.success).toHaveBeenCalledWith("success", { id: "status-id" });
    expect(toastMock.info).toHaveBeenCalledWith("info", { id: "info-id" });
  });
});

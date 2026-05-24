import { describe, expect, it, vi } from "vitest";

const { toastMock } = vi.hoisted(() => ({
  toastMock: {
    error: vi.fn(),
    success: vi.fn(),
  },
}));

vi.mock("sonner", () => ({
  toast: toastMock,
}));

import { showErrorToast, showSuccessToast } from "@/lib/toast";

describe("toast", () => {
  it("delegates each toast helper to sonner", () => {
    showErrorToast("error");
    showSuccessToast("success");

    expect(toastMock.error).toHaveBeenCalledWith("error");
    expect(toastMock.success).toHaveBeenCalledWith("success");
  });
});

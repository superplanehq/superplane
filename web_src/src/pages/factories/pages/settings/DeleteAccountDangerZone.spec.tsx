import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { DeleteAccountDangerZone } from "./DeleteAccountDangerZone";

const deleteAccount = vi.fn();

vi.mock("@/lib/accountSettings", () => ({
  deleteAccount: (...args: unknown[]) => deleteAccount(...args),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
}));

describe("DeleteAccountDangerZone", () => {
  beforeEach(() => {
    deleteAccount.mockReset();
  });

  it("requires the account email before delete", async () => {
    const user = userEvent.setup();
    deleteAccount.mockResolvedValue(undefined);
    render(<DeleteAccountDangerZone email="ada@example.com" />);

    await user.click(screen.getByTestId("account-redesign-delete"));
    const submit = screen.getByTestId("account-redesign-delete-submit");
    expect(submit).toBeDisabled();

    await user.type(screen.getByTestId("account-redesign-delete-confirm"), "wrong@example.com");
    expect(submit).toBeDisabled();

    await user.clear(screen.getByTestId("account-redesign-delete-confirm"));
    await user.type(screen.getByTestId("account-redesign-delete-confirm"), "ada@example.com");
    expect(submit).toBeEnabled();

    await user.click(submit);
    expect(deleteAccount).toHaveBeenCalledWith("ada@example.com");
  });
});

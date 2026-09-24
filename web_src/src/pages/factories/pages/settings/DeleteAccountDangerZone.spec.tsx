import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "bun:test";

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

  it("does not claim SuperPlane deletes organizations when none will be marked", async () => {
    const user = userEvent.setup();
    render(<DeleteAccountDangerZone email="ada@example.com" organizationsPendingDeletion={[]} />);

    expect(screen.getByTestId("account-redesign-danger")).not.toHaveTextContent("organizations");
    expect(screen.getByTestId("account-redesign-danger")).not.toHaveTextContent("30 days");

    await user.click(screen.getByTestId("account-redesign-delete"));
    expect(screen.queryByTestId("account-redesign-delete-orgs")).not.toBeInTheDocument();
    expect(screen.getByText("Type ada@example.com to confirm.")).toBeInTheDocument();
    expect(screen.queryByText(/marks these organizations/)).not.toBeInTheDocument();
    expect(screen.queryByText(/30 days/)).not.toBeInTheDocument();
  });

  it("lists organizations that SuperPlane will mark for deletion", async () => {
    const user = userEvent.setup();
    render(
      <DeleteAccountDangerZone
        email="ada@example.com"
        organizationsPendingDeletion={[
          { id: "org-1", name: "Acme Factory" },
          { id: "org-2", name: "Beta Labs" },
        ]}
      />,
    );

    expect(screen.getByTestId("account-redesign-danger")).toHaveTextContent("30 days");

    await user.click(screen.getByTestId("account-redesign-delete"));
    const listed = screen.getByTestId("account-redesign-delete-orgs");
    expect(listed).toHaveTextContent("Acme Factory");
    expect(listed).toHaveTextContent("Beta Labs");
    expect(screen.getByText(/marks these organizations for deletion/)).toBeInTheDocument();
  });
});

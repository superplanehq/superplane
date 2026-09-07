import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { CreateAppModal } from "./CreateAppModal";

describe("CreateAppModal", () => {
  it("does not create until the user enters a name", () => {
    const onCreate = vi.fn();

    render(<CreateAppModal open isSaving={false} onClose={vi.fn()} onCreate={onCreate} />);

    expect(screen.getByRole("dialog", { name: /create app/i })).toBeInTheDocument();
    expect(screen.getByTestId("create-app-submit-button")).toBeDisabled();
    expect(onCreate).not.toHaveBeenCalled();
  });

  it("creates with the trimmed name", async () => {
    const user = userEvent.setup();
    const onCreate = vi.fn().mockResolvedValue(undefined);

    render(<CreateAppModal open isSaving={false} onClose={vi.fn()} onCreate={onCreate} />);

    await user.type(screen.getByLabelText(/app name/i), "Sentry Analysis ");
    await user.click(screen.getByTestId("create-app-submit-button"));

    expect(onCreate).toHaveBeenCalledWith("Sentry Analysis");
  });

  it("shows an error when the name is already taken", async () => {
    const user = userEvent.setup();
    const onCreate = vi.fn().mockRejectedValue(new Error("Canvas with the same name already exists"));

    render(<CreateAppModal open isSaving={false} onClose={vi.fn()} onCreate={onCreate} />);

    await user.type(screen.getByLabelText(/app name/i), "Sentry Analysis");
    await user.click(screen.getByTestId("create-app-submit-button"));

    expect(await screen.findByText("An app with this name already exists")).toBeInTheDocument();
  });
});

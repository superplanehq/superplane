import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { useCreateSecret } from "@/hooks/useSecrets";

import { CreateSecretDialog } from ".";

vi.mock("@/hooks/useSecrets", () => ({
  useCreateSecret: vi.fn(),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

const mutateAsync = vi.fn();
const resetMutation = vi.fn();

function mockCreateSecret() {
  vi.mocked(useCreateSecret).mockReturnValue({
    error: null,
    isError: false,
    isPending: false,
    mutateAsync,
    reset: resetMutation,
  } as unknown as ReturnType<typeof useCreateSecret>);
}

describe("CreateSecretDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockCreateSecret();
  });

  it("adds and removes compact key rows", async () => {
    const user = userEvent.setup();
    render(<CreateSecretDialog open organizationId="org-123" onOpenChange={() => {}} />);

    expect(screen.getByText("1 key")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Add another key" }));
    await user.click(screen.getByRole("button", { name: "Add another key" }));

    expect(screen.getByText("3 keys")).toBeInTheDocument();
    expect(screen.getAllByTestId("secrets-create-key")).toHaveLength(3);

    await user.click(screen.getByRole("button", { name: "Remove key 2" }));

    expect(screen.getByText("2 keys")).toBeInTheDocument();
    expect(screen.getAllByTestId("secrets-create-key")).toHaveLength(2);
  });

  it("requires every visible row to be complete", async () => {
    const user = userEvent.setup();
    render(<CreateSecretDialog open organizationId="org-123" onOpenChange={() => {}} />);

    await user.type(screen.getByLabelText("Secret name"), "Production credentials");
    await user.type(screen.getByLabelText("Key name 1"), "API_TOKEN");
    await user.type(screen.getByLabelText("Secret value 1"), "secret-value");
    await user.click(screen.getByRole("button", { name: "Add another key" }));
    await user.click(screen.getByRole("button", { name: "Create secret" }));

    expect(screen.getByRole("alert")).toHaveTextContent("Enter a name and value for each key.");
    expect(mutateAsync).not.toHaveBeenCalled();
  });

  it("submits all key rows", async () => {
    const user = userEvent.setup();
    const onOpenChange = vi.fn();
    mutateAsync.mockResolvedValue({ data: { secret: { metadata: { id: "secret-1", name: "Deploy credentials" } } } });
    render(<CreateSecretDialog open organizationId="org-123" onOpenChange={onOpenChange} />);

    await user.type(screen.getByLabelText("Secret name"), "Deploy credentials");
    await user.type(screen.getByLabelText("Key name 1"), "CLIENT_ID");
    await user.type(screen.getByLabelText("Secret value 1"), "client");
    await user.click(screen.getByRole("button", { name: "Add another key" }));

    const keyInputs = screen.getAllByTestId("secrets-create-key");
    const valueInputs = screen.getAllByTestId("secrets-create-value");
    await user.type(keyInputs[1], "CLIENT_SECRET");
    await user.type(valueInputs[1], "secret");
    await user.click(screen.getByRole("button", { name: "Create secret" }));

    expect(mutateAsync).toHaveBeenCalledWith({
      name: "Deploy credentials",
      environmentVariables: [
        expect.objectContaining({ name: "CLIENT_ID", value: "client" }),
        expect.objectContaining({ name: "CLIENT_SECRET", value: "secret" }),
      ],
    });
    expect(within(screen.getByRole("dialog")).getByText("Create secret")).toBeInTheDocument();
    expect(onOpenChange).toHaveBeenCalledWith(false);
  });
});

import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { SuperplaneSecretsSecret } from "@/api-client";
import { useSecrets } from "@/hooks/useSecrets";
import { usePermissions } from "@/contexts/usePermissions";
import { SecretPickerFieldRenderer } from "./SecretPickerFieldRenderer";

vi.mock("@/hooks/useSecrets", () => ({
  useSecrets: vi.fn(),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: vi.fn(),
}));

vi.mock("@/ui/CreateSecretDialog", () => ({
  CreateSecretDialog: ({ open }: { open: boolean }) =>
    open ? <div data-testid="create-secret-dialog">Create</div> : null,
}));

function createMockSecrets(): SuperplaneSecretsSecret[] {
  return [{ metadata: { id: "1", name: "ssh-password" } }, { metadata: { id: "2", name: "github-token" } }];
}

function mockSecretsHook(options?: { isLoading?: boolean; error?: unknown; secrets?: SuperplaneSecretsSecret[] }) {
  vi.mocked(useSecrets).mockReturnValue({
    data: options?.secrets ?? createMockSecrets(),
    isLoading: options?.isLoading ?? false,
    error: options?.error ?? null,
  } as unknown as ReturnType<typeof useSecrets>);
}

function mockPermissions(canCreate: boolean) {
  vi.mocked(usePermissions).mockReturnValue({
    canAct: (resource: string, action: string) => canCreate && resource === "secrets" && action === "create",
  } as unknown as ReturnType<typeof usePermissions>);
}

function ControlledSecretPicker({ initialValue }: { initialValue: string }) {
  const [value, setValue] = React.useState<string>(initialValue);
  return (
    <SecretPickerFieldRenderer
      placeholder="Pick a secret"
      required={false}
      value={value}
      onChange={setValue}
      organizationId="org-123"
    />
  );
}

describe("SecretPickerFieldRenderer", () => {
  beforeAll(() => {
    Element.prototype.hasPointerCapture ??= () => false;
    Element.prototype.setPointerCapture ??= () => {};
    Element.prototype.releasePointerCapture ??= () => {};
    Element.prototype.scrollIntoView ??= () => {};
  });

  beforeEach(() => {
    vi.clearAllMocks();
    mockSecretsHook();
    mockPermissions(true);
  });

  it("lists the organization secrets by name", async () => {
    const user = userEvent.setup();
    render(
      <SecretPickerFieldRenderer
        placeholder="Pick a secret"
        required
        value={undefined}
        onChange={() => {}}
        organizationId="org-123"
      />,
    );

    await user.click(screen.getByRole("combobox"));

    expect(await screen.findByRole("option", { name: "ssh-password" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "github-token" })).toBeInTheDocument();
  });

  it("submits the secret name when one is selected", async () => {
    const user = userEvent.setup();
    render(<ControlledSecretPicker initialValue="" />);

    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByRole("option", { name: "ssh-password" }));

    expect(screen.getByRole("combobox")).toHaveTextContent("ssh-password");
  });

  it("offers create when the org has no secrets and the user can create", async () => {
    const user = userEvent.setup();
    mockSecretsHook({ secrets: [] });
    render(
      <SecretPickerFieldRenderer
        placeholder="Pick a secret"
        required
        value={undefined}
        onChange={() => {}}
        organizationId="org-123"
      />,
    );

    const trigger = screen.getByRole("combobox");
    expect(trigger).not.toBeDisabled();
    await user.click(trigger);
    await user.click(await screen.findByRole("option", { name: /add a new secret/i }));
    expect(screen.getByTestId("create-secret-dialog")).toBeInTheDocument();
  });

  it("disables the picker when there are no secrets and create is not allowed", () => {
    mockSecretsHook({ secrets: [] });
    mockPermissions(false);
    render(
      <SecretPickerFieldRenderer
        placeholder="Pick a secret"
        required
        value={undefined}
        onChange={() => {}}
        organizationId="org-123"
      />,
    );

    const trigger = screen.getByRole("combobox");
    expect(trigger).toBeDisabled();
    expect(screen.getByText(/Create a secret/i)).toBeInTheDocument();
  });

  it("requires an organization context", () => {
    render(
      <SecretPickerFieldRenderer
        placeholder="Pick a secret"
        required
        value={undefined}
        onChange={() => {}}
        organizationId={undefined}
      />,
    );

    expect(screen.getByText(/Select an organization first/i)).toBeInTheDocument();
  });
});

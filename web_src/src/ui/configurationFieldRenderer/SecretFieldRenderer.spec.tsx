import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import type { ConfigurationField, SuperplaneSecretsSecret } from "@/api-client";
import { useSecrets } from "@/hooks/useSecrets";
import { usePermissions } from "@/contexts/usePermissions";
import { SecretFieldRenderer, type SecretRefValue } from "./SecretFieldRenderer";

vi.mock("@/hooks/useSecrets", () => ({
  useSecrets: vi.fn(),
}));

vi.mock("@/contexts/usePermissions", () => ({
  usePermissions: vi.fn(),
}));

vi.mock("@/ui/CreateSecretDialog", () => ({
  CreateSecretDialog: ({ open, onCreated }: { open: boolean; onCreated?: (created: { name?: string }) => void }) =>
    open ? (
      <div data-testid="create-secret-dialog">
        <button type="button" onClick={() => onCreated?.({ name: "created-secret" })}>
          finish-create
        </button>
      </div>
    ) : null,
}));

function createSecrets(): SuperplaneSecretsSecret[] {
  return [{ metadata: { id: "1", name: "ssh-password" } }, { metadata: { id: "2", name: "github-token" } }];
}

function mockSecretsHook(secrets: SuperplaneSecretsSecret[] = createSecrets()) {
  vi.mocked(useSecrets).mockReturnValue({
    data: secrets,
    isLoading: false,
    error: null,
  } as unknown as ReturnType<typeof useSecrets>);
}

function mockPermissions(canCreate: boolean) {
  vi.mocked(usePermissions).mockReturnValue({
    canAct: (resource: string, action: string) => canCreate && resource === "secrets" && action === "create",
  } as unknown as ReturnType<typeof usePermissions>);
}

function createField(): ConfigurationField {
  return {
    name: "secret",
    type: "secret",
    label: "Secret",
    placeholder: "Select secret",
  };
}

function ControlledSecretFieldRenderer({ initialValue }: { initialValue: SecretRefValue }) {
  const [value, setValue] = React.useState<SecretRefValue>(initialValue);

  return (
    <SecretFieldRenderer field={createField()} isRequired value={value} onChange={setValue} organizationId="org_123" />
  );
}

describe("SecretFieldRenderer", () => {
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

  it("lists the organization secrets and stores the secret name on selection", async () => {
    const user = userEvent.setup();
    let latestValue: SecretRefValue;

    render(
      <SecretFieldRenderer
        field={createField()}
        isRequired
        value={undefined}
        onChange={(value) => {
          latestValue = value;
        }}
        organizationId="org_123"
      />,
    );

    await user.click(screen.getByRole("combobox"));
    await user.click(await screen.findByRole("option", { name: "github-token" }));

    expect(latestValue!).toEqual({ secret: "github-token" });
  });

  it("disables the picker when there are no secrets and create is not allowed", () => {
    mockSecretsHook([]);
    mockPermissions(false);

    render(
      <SecretFieldRenderer
        field={createField()}
        isRequired
        value={undefined}
        onChange={() => {}}
        organizationId="org_123"
      />,
    );

    expect(screen.getByRole("combobox")).toBeDisabled();
    expect(screen.getByText("No secrets available")).toBeInTheDocument();
    expect(screen.getByText(/Create a secret in Organization settings first/i)).toBeInTheDocument();
  });

  it("opens the create dialog when there are no secrets and selects the created secret", async () => {
    const user = userEvent.setup();
    mockSecretsHook([]);

    render(<ControlledSecretFieldRenderer initialValue={undefined} />);

    const trigger = screen.getByRole("combobox");
    expect(trigger).not.toBeDisabled();
    await user.click(trigger);
    await user.click(await screen.findByRole("option", { name: /add a new secret/i }));

    expect(await screen.findByTestId("create-secret-dialog")).toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "finish-create" }));
    expect(screen.getByRole("combobox")).toHaveTextContent("created-secret");
  });

  it("keeps listing secrets and offers add-new after a separator when secrets exist", async () => {
    const user = userEvent.setup();

    render(
      <SecretFieldRenderer
        field={createField()}
        isRequired
        value={undefined}
        onChange={() => {}}
        organizationId="org_123"
      />,
    );

    await user.click(screen.getByRole("combobox"));

    expect(await screen.findByText("ssh-password")).toBeInTheDocument();
    expect(screen.getByText("github-token")).toBeInTheDocument();
    expect(screen.getByTestId("secret-field-add-new-option")).toBeInTheDocument();
  });
});

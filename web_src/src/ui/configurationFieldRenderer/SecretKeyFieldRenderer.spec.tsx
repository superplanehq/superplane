import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import React from "react";
import { beforeAll, beforeEach, describe, expect, it, vi } from "vitest";
import { useQueries } from "@tanstack/react-query";
import type { SecretsSecret } from "@/api-client";
import { useSecrets } from "@/hooks/useSecrets";
import { SecretKeyFieldRenderer, type SecretKeyRefValue } from "./SecretKeyFieldRenderer";

vi.mock("@tanstack/react-query", () => ({
  useQueries: vi.fn(),
}));

vi.mock("@/hooks/useSecrets", () => ({
  useSecrets: vi.fn(),
  secretKeys: {
    detail: vi.fn((domainId: string, domainType: string, secretRef: string) => [domainId, domainType, secretRef]),
  },
}));

function createMockSecrets(): SecretsSecret[] {
  return [
    {
      metadata: { id: "1", name: "secret-1" },
      spec: {
        local: {
          data: {
            "api-token": "ghp_xxxxxxxxxxxxxxxxxxxx",
          },
        },
      },
    },
    {
      metadata: { id: "2", name: "secret-2" },
      spec: {
        local: {
          data: {
            token: "smp_aaaaaaaaaaaaaaaaaaa",
          },
        },
      },
    },
  ];
}

function mockSecretQueries() {
  const secrets = createMockSecrets();

  vi.mocked(useSecrets).mockReturnValue({
    data: secrets,
    isLoading: false,
    error: null,
  } as unknown as ReturnType<typeof useSecrets>);

  vi.mocked(useQueries).mockReturnValue(
    secrets.map((secret) => ({
      data: secret,
      isLoading: false,
    })) as ReturnType<typeof useQueries>,
  );
}

function ControlledSecretKeyFieldRenderer({ initialValue }: { initialValue: SecretKeyRefValue }) {
  const [value, setValue] = React.useState<SecretKeyRefValue>(initialValue);

  return (
    <SecretKeyFieldRenderer
      value={value}
      onChange={setValue}
      organizationId="org-123"
      allowClear
      placeholder="Select credential"
    />
  );
}

describe("SecretKeyFieldRenderer", () => {
  beforeAll(() => {
    Element.prototype.hasPointerCapture ??= () => false;
    Element.prototype.setPointerCapture ??= () => {};
    Element.prototype.releasePointerCapture ??= () => {};
    Element.prototype.scrollIntoView ??= () => {};
  });

  beforeEach(() => {
    vi.clearAllMocks();
    mockSecretQueries();
  });

  it("renders None as a selected value for optional empty fields", () => {
    render(
      <SecretKeyFieldRenderer
        value={undefined}
        onChange={() => {}}
        organizationId="org-123"
        allowClear
        placeholder="Select credential"
      />,
    );

    const trigger = screen.getByRole("combobox");

    expect(trigger).toHaveTextContent("None");
    expect(trigger).not.toHaveAttribute("data-placeholder");
  });

  it("keeps None styled as a selected value after clearing a credential", async () => {
    const user = userEvent.setup();

    render(<ControlledSecretKeyFieldRenderer initialValue={{ secret: "secret-1", key: "api-token" }} />);

    const trigger = screen.getByRole("combobox");
    expect(trigger).toHaveTextContent("secret-1 / api-token");

    await user.click(trigger);
    await user.click(await screen.findByText("None"));

    expect(screen.getByRole("combobox")).toHaveTextContent("None");
    expect(screen.getByRole("combobox")).not.toHaveAttribute("data-placeholder");
  });
});

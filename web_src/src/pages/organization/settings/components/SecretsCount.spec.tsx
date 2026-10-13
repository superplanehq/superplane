import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import type { SuperplaneSecretsSecret } from "@/api-client";

type SecretsQueryState = {
  data: SuperplaneSecretsSecret[] | undefined;
  isLoading: boolean;
  isError: boolean;
};

let queryState: SecretsQueryState = { data: [], isLoading: false, isError: false };
const useSecretsMock = vi.fn(() => queryState);

vi.mock("@/hooks/useSecrets", () => ({
  useSecrets: useSecretsMock,
}));

import { SecretsCount } from "./SecretsCount";

function secret(id: string): SuperplaneSecretsSecret {
  return { metadata: { id, name: `secret-${id}` } };
}

describe("SecretsCount", () => {
  beforeEach(() => {
    useSecretsMock.mockClear();
  });

  it("shows the total number of organization secrets", () => {
    queryState = { data: [secret("1"), secret("2"), secret("3")], isLoading: false, isError: false };

    render(<SecretsCount organizationId="org-1" />);

    expect(screen.getByTestId("secrets-count").textContent).toBe("3");
    expect(useSecretsMock).toHaveBeenCalledWith("org-1", "DOMAIN_TYPE_ORGANIZATION");
  });

  it("shows 0 when the organization has no secrets", () => {
    queryState = { data: [], isLoading: false, isError: false };

    render(<SecretsCount organizationId="org-1" />);

    expect(screen.getByTestId("secrets-count").textContent).toBe("0");
  });

  it("renders nothing while secrets load", () => {
    queryState = { data: undefined, isLoading: true, isError: false };

    render(<SecretsCount organizationId="org-1" />);

    expect(screen.queryByTestId("secrets-count")).toBeNull();
  });

  it("renders nothing when the secrets request fails", () => {
    queryState = { data: undefined, isLoading: false, isError: true };

    render(<SecretsCount organizationId="org-1" />);

    expect(screen.queryByTestId("secrets-count")).toBeNull();
  });
});

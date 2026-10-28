import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { peekIntegrationSetupReturn } from "@/lib/integrationSetupReturn";

import { useFirstRunGitHubEnterprise } from "./useFirstRunGitHubEnterprise";

const createMutate = vi.hoisted(() => vi.fn());
const connected = vi.hoisted(() => ({ data: [] as Array<{ metadata?: { id?: string; name?: string } }> }));

vi.mock("@/hooks/useIntegrations", () => ({
  useCreateIntegration: () => ({ mutateAsync: createMutate }),
  useConnectedIntegrations: () => connected,
}));

const WORKSPACE_GITHUB_SETUP = "/org-1/workspaces/pay-k7m2xqab/settings/organization/integrations/github/setup";
const RETURN_TO = "/org-1/workspaces/pay-k7m2xqab/setup?step=repo";
const LEGACY_INTEGRATIONS = "/org-1/settings/integrations/";

function CurrentPath() {
  const { pathname, search } = useLocation();
  return <span>{`${pathname}${search}`}</span>;
}

function Harness({ integrationId }: { integrationId?: string }) {
  const { connect, resume } = useFirstRunGitHubEnterprise({
    organizationId: "org-1",
    factoryId: "factory-1",
    factoryKey: "pay-k7m2xqab",
    integrationId,
  });
  return (
    <>
      <button type="button" onClick={() => void connect("acme.ghe.com", RETURN_TO)}>
        connect
      </button>
      <button type="button" onClick={() => resume(RETURN_TO)}>
        resume
      </button>
    </>
  );
}

function renderEnterprise(integrationId?: string) {
  render(
    <MemoryRouter initialEntries={["/org-1/workspaces/pay-k7m2xqab/setup"]}>
      <Harness integrationId={integrationId} />
      <Routes>
        <Route path="*" element={<CurrentPath />} />
      </Routes>
    </MemoryRouter>,
  );
}

describe("useFirstRunGitHubEnterprise", () => {
  beforeEach(() => {
    sessionStorage.clear();
    window.localStorage.clear();
    createMutate.mockReset();
    connected.data = [];
  });

  it("opens the workspace GitHub App setup page after create", async () => {
    const user = userEvent.setup();
    createMutate.mockResolvedValue({
      data: { integration: { metadata: { id: "int-ghe" } } },
    });
    renderEnterprise();

    await user.click(screen.getByRole("button", { name: "connect" }));

    expect(await screen.findByText(`${WORKSPACE_GITHUB_SETUP}?integrationId=int-ghe`)).toBeInTheDocument();
    expect(screen.queryByText(new RegExp(LEGACY_INTEGRATIONS))).not.toBeInTheDocument();
  });

  it("resumes a pending App on the workspace GitHub App setup page", async () => {
    const user = userEvent.setup();
    renderEnterprise("int-ghe");

    await user.click(screen.getByRole("button", { name: "resume" }));

    expect(await screen.findByText(`${WORKSPACE_GITHUB_SETUP}?integrationId=int-ghe`)).toBeInTheDocument();
    expect(screen.queryByText(new RegExp(LEGACY_INTEGRATIONS))).not.toBeInTheDocument();
    expect(peekIntegrationSetupReturn("org-1")).toBe(RETURN_TO);
  });
});

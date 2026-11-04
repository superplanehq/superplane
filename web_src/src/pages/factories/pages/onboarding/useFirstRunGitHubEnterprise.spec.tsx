import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Route, Routes, useLocation } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { peekIntegrationSetupReturn } from "@/lib/integrationSetupReturn";

import { useFirstRunGitHubEnterprise } from "./useFirstRunGitHubEnterprise";

const createMutate = vi.hoisted(() => vi.fn());
const nextStepMutate = vi.hoisted(() => vi.fn());
const openRedirectPrompt = vi.hoisted(() => vi.fn());
const connected = vi.hoisted(() => ({ data: [] as Array<{ metadata?: { id?: string; name?: string } }> }));
const described = vi.hoisted(
  () =>
    ({
      data: null as {
        status?: {
          state?: string;
          setupState?: { currentStep?: { name?: string; redirectPrompt?: { url?: string } } };
        };
      } | null,
    }) satisfies { data: unknown },
);

vi.mock("@/hooks/useIntegrations", () => ({
  useCreateIntegration: () => ({ mutateAsync: createMutate }),
  useConnectedIntegrations: () => connected,
  useNextIntegrationSetupStep: () => ({ mutateAsync: nextStepMutate }),
  useIntegration: () => described,
}));

vi.mock("@/lib/integrations", () => ({
  openRedirectPrompt: (...args: unknown[]) => openRedirectPrompt(...args),
}));

const WORKSPACE_GITHUB_SETUP = "/org-1/workspaces/pay-k7m2xqab/settings/organization/integrations/github/setup";
const RETURN_TO = "/org-1/workspaces/pay-k7m2xqab/setup?step=repo";
const LEGACY_INTEGRATIONS = "/org-1/settings/integrations/";
const CREATE_APP_STEP = {
  name: "createApp",
  redirectPrompt: { url: "https://acme.ghe.com/organizations/acme/settings/apps/new" },
};

function CurrentPath() {
  const { pathname, search } = useLocation();
  return <span>{`${pathname}${search}`}</span>;
}

function Harness({ integrationId, host }: { integrationId?: string; host?: string }) {
  const { connect, submitOwner, resume } = useFirstRunGitHubEnterprise({
    organizationId: "org-1",
    factoryId: "factory-1",
    factoryKey: "pay-k7m2xqab",
    integrationId,
    host,
  });
  return (
    <>
      <button type="button" onClick={() => void connect("acme.ghe.com", RETURN_TO)}>
        connect
      </button>
      <button type="button" onClick={() => void submitOwner("acme", RETURN_TO)}>
        submit owner
      </button>
      <button type="button" onClick={() => resume(RETURN_TO)}>
        resume
      </button>
    </>
  );
}

function renderEnterprise(integrationId?: string, host = "acme.ghe.com") {
  render(
    <MemoryRouter initialEntries={["/org-1/workspaces/pay-k7m2xqab/setup"]}>
      <Harness integrationId={integrationId} host={host} />
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
    nextStepMutate.mockReset();
    openRedirectPrompt.mockReset();
    connected.data = [];
    described.data = null;
  });

  it("creates the connection and stays on onboarding", async () => {
    const user = userEvent.setup();
    createMutate.mockResolvedValue({
      data: { integration: { metadata: { id: "int-ghe" } } },
    });
    renderEnterprise();

    await user.click(screen.getByRole("button", { name: "connect" }));

    expect(createMutate).toHaveBeenCalled();
    expect(screen.getByText("/org-1/workspaces/pay-k7m2xqab/setup")).toBeInTheDocument();
    expect(screen.queryByText(new RegExp(WORKSPACE_GITHUB_SETUP))).not.toBeInTheDocument();
    expect(screen.queryByText(new RegExp(LEGACY_INTEGRATIONS))).not.toBeInTheDocument();
  });

  it("opens GitHub from the organization step without leaving onboarding", async () => {
    const user = userEvent.setup();
    nextStepMutate.mockResolvedValue({
      data: { integration: { status: { setupState: { currentStep: CREATE_APP_STEP } } } },
    });
    renderEnterprise("int-ghe");

    await user.click(screen.getByRole("button", { name: "submit owner" }));

    expect(nextStepMutate).toHaveBeenCalledWith({
      integrationId: "int-ghe",
      inputs: {
        githubLocation: "ghe.com",
        host: "acme.ghe.com",
        ownerType: "Organization",
        owner: "acme",
      },
    });
    expect(openRedirectPrompt).toHaveBeenCalledWith(CREATE_APP_STEP);
    expect(screen.getByText("/org-1/workspaces/pay-k7m2xqab/setup")).toBeInTheDocument();
    expect(peekIntegrationSetupReturn("org-1")).toBe(RETURN_TO);
  });

  it("reopens GitHub from a pending App create step", async () => {
    const user = userEvent.setup();
    described.data = { status: { setupState: { currentStep: CREATE_APP_STEP } } };
    renderEnterprise("int-ghe");

    await user.click(screen.getByRole("button", { name: "resume" }));

    expect(openRedirectPrompt).toHaveBeenCalledWith(CREATE_APP_STEP);
    expect(screen.queryByText(new RegExp(WORKSPACE_GITHUB_SETUP))).not.toBeInTheDocument();
    expect(peekIntegrationSetupReturn("org-1")).toBe(RETURN_TO);
  });

  it("keeps a pending owner step on onboarding", async () => {
    const user = userEvent.setup();
    described.data = { status: { setupState: { currentStep: { name: "selectOwner" } } } };
    renderEnterprise("int-ghe");

    await user.click(screen.getByRole("button", { name: "resume" }));

    expect(openRedirectPrompt).not.toHaveBeenCalled();
    expect(screen.getByText("/org-1/workspaces/pay-k7m2xqab/setup")).toBeInTheDocument();
  });
});

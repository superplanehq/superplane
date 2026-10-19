import { render, screen } from "@testing-library/react";
import { afterEach, describe, expect, it, vi } from "bun:test";
import { MemoryRouter, Route, Routes, useSearchParams } from "react-router";

import { AccountContext, type AccountContextType } from "@/contexts/accountContextState";

import { SetupGuard } from "./SetupGuard";

function jsonResponse(body: unknown) {
  return new Response(JSON.stringify(body), {
    status: 200,
    headers: { "Content-Type": "application/json" },
  });
}

function account(installationAdmin: boolean): AccountContextType["account"] {
  return {
    id: "account-1",
    name: "Ada Lovelace",
    email: "ada@example.com",
    avatar_url: "",
    installation_admin: installationAdmin,
    has_password: true,
  };
}

function SetupLanding() {
  const [params] = useSearchParams();
  return <div>Setup {params.get("step")}</div>;
}

function renderGuard(initialEntry: string, value: AccountContextType) {
  return render(
    <AccountContext.Provider value={value}>
      <MemoryRouter initialEntries={[initialEntry]}>
        <SetupGuard>
          <Routes>
            <Route path="/" element={<div>Home</div>} />
            <Route path="/setup" element={<SetupLanding />} />
          </Routes>
        </SetupGuard>
      </MemoryRouter>
    </AccountContext.Provider>,
  );
}

describe("SetupGuard", () => {
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("sends an installation admin from / back to the open smtp step", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ status: "in_progress", step: "smtp" })),
    );

    renderGuard("/", {
      account: account(true),
      loading: false,
      setupRequired: false,
      refreshAccount: async () => undefined,
    });

    expect(await screen.findByText("Setup smtp")).toBeInTheDocument();
    expect(screen.queryByText("Home")).not.toBeInTheDocument();
  });

  it("keeps a member on / while owner setup is open", () => {
    renderGuard("/", {
      account: account(false),
      loading: false,
      setupRequired: false,
      refreshAccount: async () => undefined,
    });

    expect(screen.getByText("Home")).toBeInTheDocument();
  });

  it("keeps an installation admin on / after setup is closed", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(async () => jsonResponse({ status: "complete" })),
    );

    renderGuard("/", {
      account: account(true),
      loading: false,
      setupRequired: false,
      refreshAccount: async () => undefined,
    });

    expect(await screen.findByText("Home")).toBeInTheDocument();
  });

  it("still sends every visitor to /setup before an owner exists", async () => {
    renderGuard("/", {
      account: null,
      loading: false,
      setupRequired: true,
      refreshAccount: async () => undefined,
    });

    expect(await screen.findByText("Setup")).toBeInTheDocument();
  });
});

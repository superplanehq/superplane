import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { PermissionsProvider } from "@/contexts/PermissionsProvider";
import { usePermissions } from "@/contexts/usePermissions";

const accountState = { supportSession: false };

vi.mock("@/contexts/useAccount", () => ({
  useAccount: () => ({
    account: { id: "acc-1", support_session: accountState.supportSession ? { active: true } : undefined },
  }),
}));

vi.mock("@/hooks/useOrganizationId", () => ({
  useOrganizationId: () => "org-1",
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({
    data: {
      id: "user-1",
      permissions: [
        { resource: "canvases", action: "read" },
        { resource: "canvases", action: "update" },
      ],
    },
    isLoading: false,
  }),
}));

function PermissionProbe() {
  const { canAct, hasPermission } = usePermissions();
  return (
    <ul>
      <li>canAct read: {String(canAct("canvases", "read"))}</li>
      <li>canAct update: {String(canAct("canvases", "update"))}</li>
      <li>hasPermission update: {String(hasPermission("canvases", "update"))}</li>
    </ul>
  );
}

function renderProbe() {
  render(
    <PermissionsProvider>
      <PermissionProbe />
    </PermissionsProvider>,
  );
}

describe("PermissionsProvider", () => {
  beforeEach(() => {
    accountState.supportSession = false;
  });

  it("grants the permissions the user has", () => {
    renderProbe();

    expect(screen.getByText("canAct read: true")).toBeInTheDocument();
    expect(screen.getByText("canAct update: true")).toBeInTheDocument();
    expect(screen.getByText("hasPermission update: true")).toBeInTheDocument();
  });

  it("allows only read actions in a support session", () => {
    accountState.supportSession = true;

    renderProbe();

    expect(screen.getByText("canAct read: true")).toBeInTheDocument();
    expect(screen.getByText("canAct update: false")).toBeInTheDocument();
  });

  it("keeps the unmasked permission for navigation in a support session", () => {
    accountState.supportSession = true;

    renderProbe();

    expect(screen.getByText("hasPermission update: true")).toBeInTheDocument();
  });
});

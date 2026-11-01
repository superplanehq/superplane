import { render, screen } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "bun:test";
import { PermissionsProvider } from "@/contexts/PermissionsProvider";
import { usePermissions } from "@/contexts/usePermissions";

const accountState = { supportSession: false };
const organizationState = { suspended: false };
vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganization: () => ({ data: { suspended: organizationState.suspended } }),
}));

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
  const { canAct, hasPermission, canCancel } = usePermissions();
  return (
    <ul>
      <li>canAct read: {String(canAct("canvases", "read"))}</li>
      <li>canAct update: {String(canAct("canvases", "update"))}</li>
      <li>hasPermission update: {String(hasPermission("canvases", "update"))}</li>
      <li>canCancel: {String(canCancel?.("canvases"))}</li>
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
    organizationState.suspended = false;
  });

  it("grants the permissions the user has", () => {
    renderProbe();

    expect(screen.getByText("canAct read: true")).toBeInTheDocument();
    expect(screen.getByText("canAct update: true")).toBeInTheDocument();
    expect(screen.getByText("hasPermission update: true")).toBeInTheDocument();
  });

  it("keeps pages readable and cancellation available during suspension", () => {
    organizationState.suspended = true;
    renderProbe();
    expect(screen.getByText("canAct read: true")).toBeInTheDocument();
    expect(screen.getByText("canAct update: false")).toBeInTheDocument();
    expect(screen.getByText("hasPermission update: true")).toBeInTheDocument();
    expect(screen.getByText("canCancel: true")).toBeInTheDocument();
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

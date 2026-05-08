import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { MemoryRouter } from "react-router-dom";

// jsdom doesn't implement scrollIntoView, which Radix Select uses
Element.prototype.scrollIntoView = vi.fn();

const mockUseOrganizationRoles = vi.fn(() => ({ data: [], isLoading: false }));
const mockUseServiceAccounts = vi.fn(() => ({ data: [], isLoading: false }));
const mockUseCreateServiceAccount = vi.fn(() => ({
  mutateAsync: vi.fn(),
  isPending: false,
  reset: vi.fn(),
  error: null,
}));
const mockUseDeleteServiceAccount = vi.fn(() => ({
  mutateAsync: vi.fn(),
  isPending: false,
}));
const mockUsePermissions = vi.fn(() => ({ canAct: () => true, isLoading: false }));

vi.mock("@/hooks/useOrganizationData", () => ({
  useOrganizationRoles: (...args: any[]) => mockUseOrganizationRoles(...args),
}));

vi.mock("@/hooks/useServiceAccounts", () => ({
  useServiceAccounts: (...args: any[]) => mockUseServiceAccounts(...args),
  useCreateServiceAccount: (...args: any[]) => mockUseCreateServiceAccount(...args),
  useDeleteServiceAccount: (...args: any[]) => mockUseDeleteServiceAccount(...args),
}));

vi.mock("@/contexts/PermissionsContext", () => ({
  usePermissions: (...args: any[]) => mockUsePermissions(...args),
  PermissionTooltip: ({ children }: { children: React.ReactNode }) => <>{children}</>,
}));

vi.mock("@/hooks/usePageTitle", () => ({
  usePageTitle: vi.fn(),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast: vi.fn(),
  showSuccessToast: vi.fn(),
}));

import { ServiceAccounts } from "./ServiceAccounts";

function Wrapper({ children }: { children: React.ReactNode }) {
  return <MemoryRouter>{children}</MemoryRouter>;
}

describe("ServiceAccounts", () => {
  beforeEach(() => {
    mockUseOrganizationRoles.mockReturnValue({ data: [], isLoading: false });
    mockUseServiceAccounts.mockReturnValue({ data: [], isLoading: false });
    mockUseCreateServiceAccount.mockReturnValue({
      mutateAsync: vi.fn(),
      isPending: false,
      reset: vi.fn(),
      error: null,
    });
    mockUseDeleteServiceAccount.mockReturnValue({
      mutateAsync: vi.fn(),
      isPending: false,
    });
    mockUsePermissions.mockReturnValue({ canAct: () => true, isLoading: false });
  });

  it("renders without crashing", () => {
    render(<ServiceAccounts organizationId="test-org" />, { wrapper: Wrapper });
    expect(screen.getByText("Create Service Account")).toBeInTheDocument();
  });

  it("shows custom roles in the dropdown", async () => {
    mockUseOrganizationRoles.mockReturnValue({
      data: [
        {
          metadata: { name: "deployer" },
          spec: { displayName: "Deployer", description: "Can deploy to production" },
        },
        {
          metadata: { name: "org_admin" },
          spec: { displayName: "Admin", description: "Full admin access" },
        },
        {
          metadata: { name: "org_viewer" },
          spec: { displayName: "Viewer", description: "Read-only access" },
        },
      ],
      isLoading: false,
    });

    render(<ServiceAccounts organizationId="test-org" />, { wrapper: Wrapper });

    fireEvent.click(screen.getByTestId("sa-create-btn"));

    await waitFor(() => {
      expect(screen.getByTestId("sa-create-form")).toBeInTheDocument();
    });

    const roleTrigger = screen.getByTestId("sa-create-role");
    fireEvent.click(roleTrigger);

    await waitFor(() => {
      expect(screen.getByRole("option", { name: "Deployer" })).toBeInTheDocument();
    });

    expect(screen.getByRole("option", { name: "Admin" })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "Viewer" })).toBeInTheDocument();
  });

  it("shows loading state while roles are loading", async () => {
    mockUseOrganizationRoles.mockReturnValue({ data: [], isLoading: true });

    render(<ServiceAccounts organizationId="test-org" />, { wrapper: Wrapper });

    fireEvent.click(screen.getByTestId("sa-create-btn"));

    await waitFor(() => {
      expect(screen.getByText("Loading roles...")).toBeInTheDocument();
    });
  });

  it("shows empty state when no roles are available", async () => {
    mockUseOrganizationRoles.mockReturnValue({ data: [], isLoading: false });

    render(<ServiceAccounts organizationId="test-org" />, { wrapper: Wrapper });

    fireEvent.click(screen.getByTestId("sa-create-btn"));

    await waitFor(() => {
      expect(screen.getByText("No roles available")).toBeInTheDocument();
    });
  });
});

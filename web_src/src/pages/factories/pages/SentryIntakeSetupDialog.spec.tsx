import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "vitest";

import { SentryIntakeSetupDialog } from "./SentryIntakeSetupDialog";
import { SENTRY_INTAKE_SEED_SIZE, SENTRY_INTAKE_SETUP_COPY } from "./sentryIntakeSetupCopy";

const mocks = vi.hoisted(() => ({
  createIntake: vi.fn(),
  createIntegration: vi.fn(),
  issues: [] as Array<{ id: string; name: string }>,
}));

vi.mock("@/hooks/useFactoryIntakeData", () => ({
  useCreateFactoryIntake: () => ({ mutateAsync: mocks.createIntake, isPending: false }),
}));

vi.mock("@/hooks/useIntegrations", () => ({
  useConnectedIntegrations: () => ({
    data: [
      {
        metadata: { id: "integration-1", name: "Sentry", integrationName: "sentry" },
        status: { state: "ready" },
      },
    ],
    isLoading: false,
    refetch: vi.fn(),
  }),
  useAvailableIntegrations: () => ({ data: [{ name: "sentry", label: "Sentry" }] }),
  useCreateIntegration: () => ({ mutateAsync: mocks.createIntegration, reset: vi.fn() }),
  useIntegrationResources: (_organizationId: string, _integrationId: string, resourceType: string) => {
    if (resourceType === "unresolved-issue") {
      return {
        data: mocks.issues,
        isLoading: false,
        isError: false,
        refetch: vi.fn(),
      };
    }
    return {
      data: [
        { id: "payments", name: "Payments" },
        { id: "growth", name: "Growth" },
      ],
      isLoading: false,
      isError: false,
      refetch: vi.fn(),
    };
  },
}));

vi.mock("@/ui/IntegrationCreateDialog", () => ({
  IntegrationCreateDialog: ({ open, onCreated }: { open: boolean; onCreated: (id: string) => void }) =>
    open ? (
      <button type="button" data-testid="finish-connect" onClick={() => onCreated("integration-new")}>
        Finish connect
      </button>
    ) : null,
}));

function renderDialog(onCreated = vi.fn()) {
  return render(
    <MemoryRouter initialEntries={["/org-1/workspaces/sp/lines/line-plan/setup/sentry"]}>
      <SentryIntakeSetupDialog organizationId="org-1" factoryId="factory-1" onClose={vi.fn()} onCreated={onCreated} />
    </MemoryRouter>,
  );
}

describe("SentryIntakeSetupDialog", () => {
  beforeEach(() => {
    mocks.createIntake.mockReset();
    mocks.createIntake.mockResolvedValue({ id: "intake-1" });
    mocks.createIntegration.mockReset();
    mocks.createIntegration.mockResolvedValue({
      data: {
        integration: {
          metadata: { id: "integration-pending" },
          status: { browserAction: { url: "https://sentry.io/settings/" } },
        },
      },
    });
    mocks.issues.splice(0);
  });

  it("shows the 10-issue import and the listener in the wizard", async () => {
    const user = userEvent.setup();
    renderDialog();

    expect(screen.getByRole("heading", { name: SENTRY_INTAKE_SETUP_COPY.wizardStepConnect })).toBeInTheDocument();
    expect(screen.getByText(SENTRY_INTAKE_SETUP_COPY.wizardStepConnectHelper)).toBeInTheDocument();
    expect(screen.getByTestId("sentry-setup-preview-listen")).toHaveTextContent(
      SENTRY_INTAKE_SETUP_COPY.wizardPreviewListening,
    );
    expect(screen.getByTestId("sentry-setup-preview-caption")).toHaveTextContent(
      SENTRY_INTAKE_SETUP_COPY.wizardPreviewCaptionNoProject,
    );
    expect(screen.getAllByText(SENTRY_INTAKE_SETUP_COPY.wizardPreviewImporting)).toHaveLength(SENTRY_INTAKE_SEED_SIZE);

    await waitFor(() => expect(screen.getByTestId("sentry-setup-continue")).toBeEnabled());
    await user.click(screen.getByTestId("sentry-setup-continue"));

    expect(screen.getByRole("heading", { name: SENTRY_INTAKE_SETUP_COPY.wizardStepProject })).toBeInTheDocument();
    expect(screen.getByText(SENTRY_INTAKE_SETUP_COPY.wizardStepProjectHelper)).toBeInTheDocument();
    expect(screen.getByText(SENTRY_INTAKE_SETUP_COPY.wizardStepProjectHelper)).toHaveTextContent("10 newest");
    expect(screen.getByText(SENTRY_INTAKE_SETUP_COPY.wizardStepProjectHelper)).toHaveTextContent(
      "listens for new issues",
    );
  });

  it("creates a bound intake after a project is chosen", async () => {
    const user = userEvent.setup();
    const onCreated = vi.fn();
    renderDialog(onCreated);

    await waitFor(() => expect(screen.getByTestId("sentry-setup-continue")).toBeEnabled());
    await user.click(screen.getByTestId("sentry-setup-continue"));
    await user.click(screen.getByTestId("sentry-project-payments"));
    await user.click(screen.getByTestId("sentry-setup-finish"));

    await waitFor(() => {
      expect(mocks.createIntake).toHaveBeenCalledWith({
        source: "SOURCE_SENTRY_EXCEPTIONS",
        integrationId: "integration-1",
        resourceId: "payments",
      });
    });
    expect(onCreated).toHaveBeenCalled();
    expect(screen.getByTestId("sentry-setup-preview-caption")).toHaveTextContent(
      SENTRY_INTAKE_SETUP_COPY.wizardPreviewCaption,
    );
  });

  it("filters the project list by name", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByTestId("sentry-setup-continue"));
    await user.type(screen.getByLabelText("Search projects"), "grow");

    expect(screen.queryByTestId("sentry-project-payments")).not.toBeInTheDocument();
    expect(screen.getByTestId("sentry-project-growth")).toBeInTheDocument();
  });

  it("opens project selection as soon as a new organization is connected", async () => {
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByTestId("sentry-setup-connect"));
    await user.click(await screen.findByTestId("finish-connect"));

    expect(screen.getByRole("heading", { name: SENTRY_INTAKE_SETUP_COPY.wizardStepProject })).toBeInTheDocument();
    expect(screen.getByTestId("sentry-project-payments")).toBeInTheDocument();
  });
});

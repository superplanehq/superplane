import { screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, describe, expect, it, vi } from "bun:test";

import type * as CanvasDataModule from "@/hooks/useCanvasData";
import { unmockedSrc } from "@/test/unmockedModule";

import { INTAKE_CONNECTION_COPY } from "./intakeConnectionModel";
import {
  INTAKE_SETTINGS_COPY,
  intakeDangerZoneHelper,
  intakeDeleteHelper,
  intakePauseHelper,
} from "./intakeSourceSettingsModel";
import { intakeConnection, renderPopup } from "./intakeSourceSettingsPopupTestSupport";

const { useInfiniteCanvasRuns } = vi.hoisted(() => ({
  useInfiniteCanvasRuns: vi.fn(),
}));

vi.mock("@monaco-editor/react", () => ({
  Editor: ({ value, onChange }: { value?: string; onChange?: (value: string | undefined) => void }) => (
    <textarea value={value ?? ""} onChange={(event) => onChange?.(event.target.value)} />
  ),
}));

vi.mock("@/hooks/useCanvasData", () => {
  const actual = unmockedSrc<typeof CanvasDataModule>("hooks/useCanvasData");
  return {
    ...actual,
    useInfiniteCanvasRuns,
  };
});

vi.mock("@/hooks/useIntegrations", () => ({
  useIntegrationResources: () => ({
    data: [
      { id: "todo", name: "To Do" },
      { id: "qa", name: "QA" },
      { id: "done", name: "Done" },
    ],
    isLoading: false,
    isError: false,
  }),
}));

useInfiniteCanvasRuns.mockReturnValue({
  data: {
    pages: [
      {
        runs: [
          {
            id: "run-1",
            canvasId: "app-github-issues-intake",
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            createdAt: "2026-05-01T12:00:00Z",
            rootEvent: { nodeId: "trigger", customName: "feat: Add console empty-state" },
          },
        ],
      },
    ],
  },
  isPending: false,
  isError: false,
  hasNextPage: false,
  isFetchingNextPage: false,
  fetchNextPage: vi.fn(),
  refetch: vi.fn(),
});

afterEach(() => {
  localStorage.clear();
});

describe("IntakeSourceSettingsPopup connection and lifecycle", () => {
  it("hides the Connection section for a GitHub intake", () => {
    renderPopup();

    expect(screen.queryByTestId("intake-connection")).not.toBeInTheDocument();
  });

  it("shows Connection fields for a Jira intake that needs a live connection", () => {
    renderPopup({
      sourceId: "jira-issues",
      organizationId: "org-1",
      connection: intakeConnection({ health: "HEALTH_MISSING_INTEGRATION" }),
    });

    expect(screen.getByTestId("intake-connection")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: INTAKE_CONNECTION_COPY.section })).toBeInTheDocument();
    expect(screen.getByTestId("intake-connection-banner")).toHaveTextContent(INTAKE_CONNECTION_COPY.missing);
    expect(screen.getByText(INTAKE_CONNECTION_COPY.integration)).toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Atlassian/ })).toHaveAttribute(
      "href",
      "/org-1/workspaces/rf/settings/organization/integrations/jira-1",
    );
  });

  it("hides Connect when a Jira intake already has an account", () => {
    renderPopup({
      sourceId: "jira-issues",
      organizationId: "org-1",
      connection: intakeConnection({
        binding: { integrationId: "jira-1", resourceId: "ENG" },
        projects: [{ id: "ENG", name: "Engineering" }],
      }),
    });

    expect(screen.queryByTestId("intake-connection-connect")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Atlassian/ })).toHaveAttribute(
      "href",
      "/org-1/workspaces/rf/settings/organization/integrations/jira-1",
    );
  });

  it("shows Connect Jira when the intake has no account", () => {
    renderPopup({
      sourceId: "jira-issues",
      connection: intakeConnection({ integrations: [] }),
    });

    expect(screen.getByTestId("intake-connection-connect")).toHaveTextContent("Connect Jira");
  });

  it("hides Connect when a Sentry intake already has an account", () => {
    renderPopup({
      sourceId: "sentry-exceptions",
      organizationId: "org-1",
      connection: intakeConnection({
        binding: { integrationId: "sentry-1", resourceId: "proj-1" },
        integrations: [
          {
            metadata: { id: "sentry-1", name: "Sentry org", integrationName: "sentry" },
            status: { state: "ready" },
          },
        ],
        projects: [{ id: "proj-1", name: "Frontend" }],
      }),
    });

    expect(screen.queryByTestId("intake-connection-connect")).not.toBeInTheDocument();
    expect(screen.getByRole("link", { name: /Sentry org/ })).toHaveAttribute(
      "href",
      "/org-1/workspaces/rf/settings/organization/integrations/sentry-1",
    );
  });

  it("keeps Save disabled until the Jira project is chosen", () => {
    renderPopup({
      sourceId: "jira-issues",
      connection: intakeConnection({
        health: "HEALTH_MISSING_INTEGRATION",
        binding: { integrationId: "jira-1", resourceId: "" },
        projects: [{ id: "ENG", name: "Engineering" }],
        saveDisabled: true,
      }),
    });

    expect(
      within(screen.getByTestId("intake-settings-topbar")).getByTestId("intake-source-settings-save"),
    ).toBeDisabled();
  });

  it.each(["github-issues", "sentry-exceptions", "jira-issues", "productive-tasks"] as const)(
    "pauses, resumes, and deletes a %s intake after confirmation",
    async (sourceId) => {
      const onPause = vi.fn();
      const onResume = vi.fn();
      const onDelete = vi.fn();
      const user = userEvent.setup();
      renderPopup({ sourceId, onPause, onResume, onDelete });

      expect(screen.getByTestId("intake-source-settings-pause")).toHaveTextContent(INTAKE_SETTINGS_COPY.pause);
      expect(screen.getByText(intakeDangerZoneHelper(sourceId))).toBeInTheDocument();
      expect(screen.getByText(intakePauseHelper(sourceId))).toBeInTheDocument();
      expect(screen.getByText(intakeDeleteHelper(sourceId))).toBeInTheDocument();
      expect(screen.getByTestId("intake-source-settings-delete")).toHaveTextContent(INTAKE_SETTINGS_COPY.delete);

      await user.click(screen.getByTestId("intake-source-settings-pause"));
      expect(onPause).toHaveBeenCalledTimes(1);

      await user.click(screen.getByTestId("intake-source-settings-delete"));
      expect(screen.getByTestId("intake-delete-dialog")).toBeInTheDocument();
      expect(screen.getByTestId("intake-delete-dialog")).toHaveTextContent(intakeDeleteHelper(sourceId));
      expect(onDelete).not.toHaveBeenCalled();
      await user.click(screen.getByTestId("intake-delete-cancel"));
      expect(screen.queryByTestId("intake-delete-dialog")).not.toBeInTheDocument();
      expect(onDelete).not.toHaveBeenCalled();

      await user.click(screen.getByTestId("intake-source-settings-delete"));
      await user.click(screen.getByTestId("intake-delete-confirm"));
      expect(onDelete).toHaveBeenCalledTimes(1);
    },
  );

  it.each(["github-issues", "sentry-exceptions", "jira-issues", "productive-tasks"] as const)(
    "offers resume for a paused %s intake",
    async (sourceId) => {
      const onResume = vi.fn();
      const user = userEvent.setup();
      renderPopup({ sourceId, paused: true, onResume });

      expect(screen.queryByTestId("intake-source-settings-pause")).not.toBeInTheDocument();
      await user.click(screen.getByTestId("intake-source-settings-resume"));
      expect(onResume).toHaveBeenCalledTimes(1);
    },
  );

  it("keeps a failed pause from rejecting and shows the error", async () => {
    const onPause = vi.fn().mockRejectedValue(new Error("pause failed"));
    const user = userEvent.setup();
    renderPopup({
      sourceId: "sentry-exceptions",
      onPause,
      pauseError: INTAKE_SETTINGS_COPY.pauseError,
    });

    await user.click(screen.getByTestId("intake-source-settings-pause"));

    expect(onPause).toHaveBeenCalledTimes(1);
    expect(screen.getByRole("alert")).toHaveTextContent(INTAKE_SETTINGS_COPY.pauseError);
  });

  it("shows a delete error in the confirmation dialog", async () => {
    const user = userEvent.setup();
    renderPopup({
      sourceId: "sentry-exceptions",
      onDelete: vi.fn().mockRejectedValue(new Error("delete failed")),
      deleteError: INTAKE_SETTINGS_COPY.deleteError,
    });

    await user.click(screen.getByTestId("intake-source-settings-delete"));

    const dialog = screen.getByTestId("intake-delete-dialog");
    expect(within(dialog).getByTestId("intake-delete-error")).toHaveTextContent(INTAKE_SETTINGS_COPY.deleteError);
    expect(within(screen.getByTestId("intake-source-settings")).queryByRole("alert")).not.toBeInTheDocument();
  });
});

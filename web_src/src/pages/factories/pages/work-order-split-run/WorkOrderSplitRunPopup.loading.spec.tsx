import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { fireEvent, render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { beforeEach, describe, expect, it, vi } from "bun:test";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import type * as FactoryData from "@/hooks/useFactoryData";
import { unmockedSrc } from "@/test/unmockedModule";
import { TooltipProvider } from "@/ui/tooltip";
import { canvasKeys } from "@/hooks/useCanvasData";

import { APPROVAL_WORK_ORDER, DRAFT_WORK_ORDER, REFUND_FACTORY_LINES } from "../../__fixtures__/factoryPageResponses";
import { INTENT_DOCUMENT_TITLE } from "../../lib/intentDocument";
import { WorkOrderSplitRunPopup } from "./WorkOrderSplitRunPopup";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

const lookupState = vi.hoisted(() => ({
  factoryPending: false,
  planning: { enabled: true, clarity: true, confidence: true },
  sessionLoading: false,
  queryError: null as Error | null,
  artifactsLoading: false,
  artifactsError: null as Error | null,
  ready: false,
}));

vi.mock("@/hooks/useFactoryData", () => {
  const actual = unmockedSrc<typeof FactoryData>("hooks/useFactoryData");
  return {
    ...actual,
    useFactory: () => ({
      data: lookupState.factoryPending
        ? undefined
        : { id: "factory-1", planning: lookupState.planning, lines: REFUND_FACTORY_LINES },
      isPending: lookupState.factoryPending,
    }),
  };
});

vi.mock("@monaco-editor/react", () => ({
  default: ({ value }: { value?: string }) => <pre data-testid="monaco-stub">{value}</pre>,
}));

vi.mock("@/hooks/useFactoryLineRunnerModels", () => ({
  useFactoryLineRunnerModels: () => ({
    data: { models: [{ id: "task-model", name: "Task model" }] },
    isLoading: false,
  }),
}));

vi.mock("@/hooks/useExperimentalFeature", () => ({
  useExperimentalFeature: () => ({
    has: () => true,
    enabledExperimentalFeatures: [],
    isLoading: false,
  }),
}));

vi.mock("./useAnalysisPlanningSession", () => ({
  useAnalysisPlanningSession: () => ({
    organizationId: "organization-1",
    factoryId: "factory-1",
    session: null,
    isLoading: lookupState.sessionLoading,
    queryError: lookupState.queryError,
    view: { machineStatus: "waiting", messages: [], executionId: "", canvasId: "" },
    composer: "",
    canSend: false,
    onComposerChange: () => undefined,
    onSend: async () => false,
    onSubmitSurvey: () => undefined,
  }),
}));

vi.mock("./useSplitRunPopupData", () => ({
  useSplitRunPopupData: () => ({
    artifacts: lookupState.ready
      ? [
          {
            id: "plan",
            type: "TYPE_MARKDOWN",
            data: { name: "plan.md", body: "# Plan\n\nImplement the change." },
          },
        ]
      : [],
    pullRequests: [],
    sourceDescription: "",
    useLive: true,
    artifactsLoading: lookupState.artifactsLoading,
    artifactsError: lookupState.artifactsError,
    pullRequestsLoading: false,
    pullRequestsError: null,
  }),
}));

function renderPopup(
  onClose?: () => void,
  fixture = splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER),
  onDispatch?: (model?: string, thinking?: string) => Promise<void>,
) {
  const client = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  client.setQueryData(canvasKeys.detail("organization-1", "app-refund-implementer"), {
    spec: {
      nodes: [
        {
          id: "implementation-agent-no-issue",
          component: "runnerOpenRouter",
          configuration: { model: "openai/gpt-6.1-sol", thinkingLevel: "high" },
        },
      ],
    },
  });
  render(
    <QueryClientProvider client={client}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <WorkOrderSplitRunPopup
              organizationId="organization-1"
              factoryId="factory-1"
              orderId="work-order-1"
              lineId={REFUND_FACTORY_LINES[0].id}
              canDispatch
              onDispatch={onDispatch}
              fixture={fixture}
              onClose={onClose}
            />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
  return client;
}

describe("WorkOrderSplitRunPopup loading mode", () => {
  beforeEach(() => {
    lookupState.factoryPending = false;
    lookupState.planning = { enabled: true, clarity: true, confidence: true };
    lookupState.sessionLoading = false;
    lookupState.queryError = null;
    lookupState.artifactsLoading = false;
    lookupState.artifactsError = null;
    lookupState.ready = false;
  });

  it("starts an untouched task without overriding saved High thinking", async () => {
    const onDispatch = vi.fn(async () => {});
    lookupState.ready = true;
    renderPopup(undefined, splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER), onDispatch);
    expect(screen.getByTestId("split-run-draft-model")).toHaveAccessibleName(/openai\/gpt-6.1-sol.*High.*Default/);
    await userEvent.setup().click(screen.getByTestId("split-run-draft-model"));
    await userEvent.setup().keyboard("{Escape}");
    await userEvent.setup().click(screen.getByRole("button", { name: "Start" }));
    await userEvent.setup().click(screen.getByRole("button", { name: "Start anyway" }));
    expect(onDispatch).toHaveBeenCalledWith(undefined, undefined);
  });

  it.each(["model", "thinking"])(
    "dispatches only an explicit %s override while saved defaults refresh",
    async (field) => {
      lookupState.ready = true;
      const user = userEvent.setup();
      const onDispatch = vi.fn(async () => {});
      const client = renderPopup(undefined, splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER), onDispatch);
      const trigger = screen.getByTestId("split-run-draft-model");
      await user.click(trigger);
      await user.hover(
        screen.getByTestId(field === "model" ? "split-run-draft-model-list" : "split-run-draft-thinking"),
      );
      fireEvent.click(await screen.findByRole("menuitem", { name: field === "model" ? "Task model" : "High" }));
      await user.keyboard("{Escape}");
      client.setQueryData(canvasKeys.detail("organization-1", "app-refund-implementer"), {
        spec: {
          nodes: [{ component: "runnerOpenRouter", configuration: { model: "new-default", thinkingLevel: "low" } }],
        },
      });
      await waitFor(() =>
        expect(trigger).toHaveAccessibleName(
          field === "model" ? /Task model.*Low \(Default\)/ : /new-default.*Default.*High$/,
        ),
      );
      await user.click(screen.getByRole("button", { name: "Start" }));
      await user.click(screen.getByRole("button", { name: "Start anyway" }));
      expect(onDispatch).toHaveBeenLastCalledWith(
        field === "model" ? "task-model" : undefined,
        field === "thinking" ? "high" : undefined,
      );
      await user.click(trigger);
      await user.click(screen.getByRole("menuitem", { name: "Use defaults" }));
      await user.click(screen.getByRole("button", { name: "Start" }));
      await user.click(screen.getByRole("button", { name: "Start anyway" }));
      expect(onDispatch).toHaveBeenLastCalledWith(undefined, undefined);
    },
  );

  it("shows a dismissible loading popup while Planning settings load", async () => {
    const user = userEvent.setup();
    const onClose = vi.fn();
    lookupState.factoryPending = true;

    renderPopup(onClose);

    expect(screen.getByTestId("work-order-split-run-loading")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "RF-105 Draft: rework refund telemetry" })).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Loading task");
    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();

    await user.click(screen.getByRole("button", { name: "Close" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("shows a loading popup while the refinement session loads", () => {
    lookupState.sessionLoading = true;

    renderPopup();

    expect(screen.getByTestId("work-order-split-run-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
  });

  it("shows a loading popup while draft artifacts load", () => {
    lookupState.artifactsLoading = true;

    renderPopup();

    expect(screen.getByTestId("work-order-split-run-loading")).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
  });

  it("keeps a started task on the analysis popup while artifacts load", () => {
    lookupState.artifactsLoading = true;
    const fixture = splitRunFixtureForWorkOrder(APPROVAL_WORK_ORDER);
    expect(fixture.footer.kind).not.toBe("draft");

    renderPopup(undefined, fixture);

    expect(screen.queryByTestId("work-order-split-run-loading")).not.toBeInTheDocument();
    expect(screen.getByTestId("work-order-split-run")).toBeInTheDocument();
    expect(screen.getByTestId("redesign-console-variant")).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: INTENT_DOCUMENT_TITLE })).not.toBeInTheDocument();
    expect(screen.queryByTestId("split-run-intent-summary")).not.toBeInTheDocument();
  });

  it("keeps a started task on the analysis popup when Planning is off", () => {
    lookupState.planning = { enabled: false, clarity: true, confidence: true };
    const fixture = splitRunFixtureForWorkOrder(APPROVAL_WORK_ORDER);
    expect(fixture.footer.kind).not.toBe("draft");

    renderPopup(undefined, fixture);

    expect(screen.getByTestId("work-order-split-run")).toBeInTheDocument();
  });

  it("uses the analysis popup for a draft when Planning is off", () => {
    lookupState.planning = { enabled: false, clarity: true, confidence: true };

    renderPopup();

    expect(screen.getByTestId("work-order-split-run")).toBeInTheDocument();
  });

  it("keeps a draft on the analysis popup when artifact lookup fails", () => {
    lookupState.artifactsError = new Error("artifacts unavailable");

    renderPopup();

    expect(screen.queryByTestId("work-order-split-run-loading")).not.toBeInTheDocument();
    expect(screen.getByTestId("work-order-split-run")).toBeInTheDocument();
  });

  it("shows a recovery alert when the refinement session lookup fails", () => {
    lookupState.queryError = new Error("session unavailable");

    renderPopup();

    expect(screen.getByTestId("work-order-split-run")).toBeInTheDocument();
    expect(screen.getByRole("alert")).toHaveTextContent(
      "The refinement session did not load. Refresh the page to try again.",
    );
  });
});

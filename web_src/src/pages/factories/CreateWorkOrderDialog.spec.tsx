import { act, cleanup, render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import {
  EMPTY_FACTORY,
  PRIMARY_FACTORY_ID,
  PRIMARY_FACTORY_KEY,
  REFUND_FACTORY,
  factoryWithPlanning,
} from "./__fixtures__/factoryPageResponses";
import { CreateWorkOrderDialog } from "./CreateWorkOrderDialog";
import { CREATE_WORK_ORDER_REQUEST_COPY } from "./createWorkOrderRequestCopy";
import { FactoriesLayoutContext } from "./layout/factoriesLayoutContext";

const { createMutate, dispatchMutate, meUser, showErrorToast } = vi.hoisted(() => ({
  createMutate: vi.fn(),
  dispatchMutate: vi.fn(),
  meUser: { current: null as { id: string; name: string } | null },
  showErrorToast: vi.fn(),
}));

vi.mock("@/hooks/useFactoryData", () => ({
  useCreateWorkOrder: () => ({ mutateAsync: createMutate, isPending: false }),
  useDispatchWorkOrder: () => ({ mutateAsync: dispatchMutate, isPending: false }),
}));

vi.mock("@/hooks/useMe", () => ({
  useMe: () => ({ data: meUser.current }),
}));

vi.mock("@/lib/toast", () => ({
  showErrorToast,
}));

vi.mock("./WorkOrderDescriptionEditor", () => ({
  WorkOrderDescriptionEditor: ({
    factoryId,
    value,
    onChange,
    onFocus,
  }: {
    factoryId?: string;
    value?: string;
    onChange?: (next: string) => void;
    onFocus?: () => void;
  }) => (
    <textarea
      data-testid="work-order-description-input"
      data-factory-id={factoryId}
      value={value}
      onChange={(event) => onChange?.(event.target.value)}
      onFocus={() => onFocus?.()}
    />
  ),
}));

type ResultEvent = {
  resultIndex: number;
  results: Array<{ isFinal: boolean; 0: { transcript: string } }>;
};

class FakeSpeechRecognition {
  static instances: FakeSpeechRecognition[] = [];

  continuous = false;
  interimResults = false;
  lang = "";
  onresult: ((event: ResultEvent) => void) | null = null;
  onerror: ((event: { error: string }) => void) | null = null;
  onend: (() => void) | null = null;
  start = vi.fn();
  stop = vi.fn();
  abort = vi.fn();

  constructor() {
    FakeSpeechRecognition.instances.push(this);
  }
}

function latestRecognition(): FakeSpeechRecognition {
  const recognition = FakeSpeechRecognition.instances.at(-1);
  if (!recognition) {
    throw new Error("Speech recognition was not created");
  }
  return recognition;
}

function emitTranscript(transcript: string, isFinal: boolean) {
  latestRecognition().onresult?.({
    resultIndex: 0,
    results: [Object.assign([{ transcript }], { isFinal, 0: { transcript } })],
  });
}

function emitFinalPhrases(transcripts: string[]) {
  latestRecognition().onresult?.({
    resultIndex: 0,
    results: transcripts.map((transcript) => Object.assign([{ transcript }], { isFinal: true, 0: { transcript } })),
  });
}

function renderDialog(
  factory = factoryWithPlanning(REFUND_FACTORY, { enabled: false, clarity: true, confidence: true }),
) {
  return render(
    <FactoriesLayoutContext.Provider
      value={{
        organizationId: "org-1",
        factoryId: PRIMARY_FACTORY_ID,
        factoryKey: PRIMARY_FACTORY_KEY,
        factory,
        factories: [factory],
        openCreateWorkOrder: vi.fn(),
      }}
    >
      <CreateWorkOrderDialog open onClose={vi.fn()} onCreated={vi.fn()} />
    </FactoriesLayoutContext.Provider>,
  );
}

describe("CreateWorkOrderDialog", () => {
  beforeEach(() => {
    createMutate.mockReset();
    dispatchMutate.mockReset();
    showErrorToast.mockReset();
    meUser.current = null;
    FakeSpeechRecognition.instances = [];
  });

  afterEach(async () => {
    // The dialog mounts a Radix focus scope that schedules a `setTimeout(0)` on
    // unmount to dispatch its "auto focus on unmount" event. Unmount here and
    // flush that timer while jsdom is still alive; otherwise it fires during
    // environment teardown and throws an unhandled "dispatchEvent" TypeError
    // that fails the whole test shard even though every test passed.
    cleanup();
    vi.unstubAllGlobals();
    await act(async () => {
      await new Promise((resolve) => setTimeout(resolve, 0));
    });
  });

  it("names the dialog New task instead of the fallback Dialog title", () => {
    renderDialog();

    expect(screen.getByRole("dialog", { name: "New task" })).toBeInTheDocument();
    expect(screen.queryByText("Dialog")).not.toBeInTheDocument();
  });

  it("keeps only expand and close controls in the header", () => {
    renderDialog();

    const header = screen.getByTestId("work-order-create-header");
    expect(within(header).getByTestId("work-order-create-fullscreen-button")).toBeInTheDocument();
    expect(within(header).queryByTestId("work-order-create-button")).not.toBeInTheDocument();
  });

  it("renders a single Create button in the footer, with no line picker", () => {
    renderDialog();

    expect(screen.getByTestId("work-order-create-button")).toHaveTextContent("Create");
    expect(screen.queryByTestId("work-order-create-start")).not.toBeInTheDocument();
    expect(screen.queryByTestId("work-order-line-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("work-order-line-picker-panel")).not.toBeInTheDocument();
  });

  it("does not show an owner picker", () => {
    meUser.current = { id: "user-me", name: "Me" };
    renderDialog();

    expect(screen.queryByTestId("work-order-assignees-button")).not.toBeInTheDocument();
    expect(screen.queryByTestId("mock-owner-pill")).not.toBeInTheDocument();
  });

  it("creates the task without sending it to a line", async () => {
    const user = userEvent.setup();
    createMutate.mockResolvedValue({ id: "order-1", number: "101" });
    renderDialog();

    await user.type(screen.getByTestId("work-order-title-input"), "Ship the refunds line");
    await user.click(screen.getByTestId("work-order-create-button"));

    expect(createMutate).toHaveBeenCalled();
    expect(dispatchMutate).not.toHaveBeenCalled();
  });

  it("keeps Create enabled when the workspace has no lines", async () => {
    renderDialog(factoryWithPlanning(EMPTY_FACTORY, { enabled: false, clarity: true, confidence: true }));

    await userEvent.setup().type(screen.getByTestId("work-order-title-input"), "Draft only");

    expect(screen.getByTestId("work-order-create-button")).not.toBeDisabled();
  });

  it("opens the request composer when Planning is on", async () => {
    const user = userEvent.setup();
    createMutate.mockResolvedValue({ id: "order-1", number: "101" });
    renderDialog(REFUND_FACTORY);

    expect(screen.getByTestId("create-work-order-request-dialog")).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-title-input")).not.toBeInTheDocument();
    expect(screen.getByRole("dialog", { name: CREATE_WORK_ORDER_REQUEST_COPY.title })).toBeInTheDocument();
    expect(screen.getByTestId("work-order-description-input")).toHaveAttribute("data-factory-id", PRIMARY_FACTORY_ID);

    await user.type(screen.getByTestId("work-order-description-input"), "Refunds fail on retry.");
    await user.click(screen.getByTestId("create-work-order-request-create"));

    expect(createMutate).toHaveBeenCalledWith({
      title: "Refunds fail on retry.",
      description: "Refunds fail on retry.",
      assigneeIds: [],
    });
    expect(dispatchMutate).not.toHaveBeenCalled();
  });

  it("hides the dictate button when speech recognition is missing", () => {
    renderDialog();

    expect(screen.queryByTestId("dictate-button")).not.toBeInTheDocument();
  });

  it("shows the dictate button and does not listen until click", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    renderDialog();

    expect(screen.getByTestId("dictate-button")).toHaveAccessibleName(CREATE_WORK_ORDER_REQUEST_COPY.dictate);
    expect(screen.getByTestId("dictate-mic-icon")).toBeInTheDocument();
    expect(screen.queryByTestId("dictate-stop-icon")).not.toBeInTheDocument();
    expect(screen.getByTestId("dictate-button")).not.toHaveClass(
      "text-destructive",
      "bg-destructive/15",
      "ring-destructive",
    );
    expect(FakeSpeechRecognition.instances).toHaveLength(0);

    await user.click(screen.getByTestId("dictate-button"));

    expect(latestRecognition().start).toHaveBeenCalledTimes(1);
    expect(screen.getByTestId("dictate-button")).toHaveAccessibleName(CREATE_WORK_ORDER_REQUEST_COPY.stopDictation);
    expect(screen.getByTestId("dictate-button")).toHaveAttribute("aria-pressed", "true");
    expect(screen.getByTestId("dictate-stop-icon")).toBeInTheDocument();
    expect(screen.queryByTestId("dictate-mic-icon")).not.toBeInTheDocument();
    expect(screen.getByTestId("dictate-button")).toHaveClass(
      "text-destructive",
      "bg-destructive/15",
      "ring-2",
      "ring-destructive",
      "animate-pulse",
      "hover:bg-destructive/15",
      "hover:text-destructive",
      "dark:hover:bg-destructive/15",
      "dark:hover:text-destructive",
      "motion-reduce:animate-none",
    );

    await user.click(screen.getByTestId("dictate-button"));

    expect(latestRecognition().abort).toHaveBeenCalled();
    expect(screen.getByTestId("dictate-button")).toHaveAccessibleName(CREATE_WORK_ORDER_REQUEST_COPY.dictate);
    expect(screen.getByTestId("dictate-mic-icon")).toBeInTheDocument();
    expect(screen.queryByTestId("dictate-stop-icon")).not.toBeInTheDocument();
    expect(screen.getByTestId("dictate-button")).not.toHaveClass(
      "text-destructive",
      "bg-destructive/15",
      "ring-destructive",
    );
  });

  it("writes the live phrase into the description and keeps the toolbar still", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      emitTranscript("Fix refunds", false);
    });

    expect(screen.queryByTestId("dictate-interim")).not.toBeInTheDocument();
    expect(screen.getByTestId("work-order-title-input")).toHaveValue("Fix refunds");
    expect(screen.getByTestId("work-order-description-input")).toHaveValue("");
  });

  it("appends a final phrase to the title when the title was last focused", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByTestId("work-order-title-input"));
    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      emitTranscript("Fix refunds", true);
    });

    expect(screen.getByTestId("work-order-title-input")).toHaveValue("Fix refunds");
    expect(screen.getByTestId("work-order-description-input")).toHaveValue("");
  });

  it("appends a final phrase to the description when the title is not focused", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByTestId("work-order-description-input"));
    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      emitTranscript("Retry fails", true);
    });

    expect(screen.getByTestId("work-order-description-input")).toHaveValue("Retry fails");
    expect(screen.getByTestId("work-order-title-input")).toHaveValue("");
  });

  it("inserts a space before a final phrase when the field already has text", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    renderDialog();

    await user.type(screen.getByTestId("work-order-title-input"), "Fix");
    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      emitTranscript("refunds", true);
    });

    expect(screen.getByTestId("work-order-title-input")).toHaveValue("Fix refunds");
  });

  it("keeps every final phrase from one recognition event", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByTestId("work-order-title-input"));
    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      emitFinalPhrases(["Fix refunds", "on retry"]);
    });

    expect(screen.getByTestId("work-order-title-input")).toHaveValue("Fix refunds on retry");
  });

  it("truncates a dictated title to 256 characters", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByTestId("work-order-title-input"));
    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      emitTranscript("A".repeat(300), true);
    });

    expect(screen.getByTestId("work-order-title-input")).toHaveValue("A".repeat(256));
  });

  it("stops dictation when the dialog closes", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByTestId("dictate-button"));
    const recognition = latestRecognition();
    await user.click(screen.getByTestId("work-order-create-close-button"));

    expect(recognition.abort).toHaveBeenCalled();
  });

  it("stops dictation before create", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    createMutate.mockResolvedValue({ id: "order-1", number: "101" });
    renderDialog();

    await user.type(screen.getByTestId("work-order-title-input"), "Ship the refunds line");
    await user.click(screen.getByTestId("dictate-button"));
    const recognition = latestRecognition();
    await user.click(screen.getByTestId("work-order-create-button"));

    expect(recognition.abort).toHaveBeenCalled();
    expect(createMutate).toHaveBeenCalled();
  });

  it("shows a permission error toast and returns to idle", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    renderDialog();

    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      latestRecognition().onerror?.({ error: "not-allowed" });
    });

    expect(showErrorToast).toHaveBeenCalledWith(CREATE_WORK_ORDER_REQUEST_COPY.microphoneDenied);
    expect(screen.getByTestId("dictate-button")).toHaveAttribute("aria-pressed", "false");
    expect(screen.getByTestId("dictate-button")).toHaveAccessibleName(CREATE_WORK_ORDER_REQUEST_COPY.dictate);
  });
});

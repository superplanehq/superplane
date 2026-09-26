import { act, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  analysisChat,
  INTENT,
  INTENT_DOC,
  IntentDocumentResizeObserver,
  renderIntentDocument,
  WAITING_COMPOSER_VIEW,
} from "./WorkOrderIntentDocument.testHelpers";
import { WorkOrderIntentDocument } from "./WorkOrderIntentDocument";
import { ANALYSIS_PLANNING_COPY } from "./useAnalysisPlanningSession";
import { resetStreamMemoryForTests } from "./useStreamOnUpdate";

const { showErrorToast } = vi.hoisted(() => ({ showErrorToast: vi.fn() }));

vi.mock("@/lib/toast", () => ({ showErrorToast }));

vi.mock("@/hooks/useOrgUserLookup", () => ({
  useOrgUserLookup: () => ({
    resolveUser: (id: string | undefined, name?: string) =>
      id ? { id, name: name ?? "Ada Lovelace", initials: "AL" } : null,
    isLoading: false,
  }),
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

describe("WorkOrderIntentDocument composer dictation", () => {
  beforeEach(() => {
    window.localStorage.clear();
    FakeSpeechRecognition.instances = [];
    vi.stubGlobal("ResizeObserver", IntentDocumentResizeObserver);
  });

  afterEach(() => {
    window.localStorage.clear();
    vi.unstubAllGlobals();
    resetStreamMemoryForTests();
    showErrorToast.mockReset();
  });

  it("hides the dictate button when speech recognition is missing", () => {
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        analysis={analysisChat({ view: WAITING_COMPOSER_VIEW })}
      />,
    );

    expect(screen.queryByTestId("dictate-button")).not.toBeInTheDocument();
  });

  it("shows the dictate button and does not listen until click", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        analysis={analysisChat({ view: WAITING_COMPOSER_VIEW })}
      />,
    );

    expect(screen.getByTestId("dictate-button")).toHaveAccessibleName(ANALYSIS_PLANNING_COPY.dictate);
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
    expect(screen.getByTestId("dictate-button")).toHaveAccessibleName(ANALYSIS_PLANNING_COPY.stopDictation);
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
    expect(screen.getByTestId("dictate-button")).toHaveAccessibleName(ANALYSIS_PLANNING_COPY.dictate);
    expect(screen.getByTestId("dictate-mic-icon")).toBeInTheDocument();
    expect(screen.queryByTestId("dictate-stop-icon")).not.toBeInTheDocument();
    expect(screen.getByTestId("dictate-button")).not.toHaveClass(
      "text-destructive",
      "bg-destructive/15",
      "ring-destructive",
    );
  });

  it("writes the live phrase into the composer and keeps the toolbar still", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    const onComposerChange = vi.fn();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        analysis={analysisChat({ view: WAITING_COMPOSER_VIEW, onComposerChange })}
      />,
    );

    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      emitTranscript("Need the empty state", false);
    });

    expect(screen.queryByTestId("dictate-interim")).not.toBeInTheDocument();
    expect(onComposerChange).toHaveBeenCalledWith("Need the empty state");
  });

  it("appends a final phrase to the composer", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    const onComposerChange = vi.fn();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        analysis={analysisChat({
          view: WAITING_COMPOSER_VIEW,
          composer: "Need the empty state.",
          onComposerChange,
        })}
      />,
    );

    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      emitTranscript("Confirm the copy", true);
    });

    expect(onComposerChange).toHaveBeenCalledWith("Need the empty state. Confirm the copy");
  });

  it("keeps every final phrase from one recognition event", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    const onComposerChange = vi.fn();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        analysis={analysisChat({
          view: WAITING_COMPOSER_VIEW,
          composer: "Need the empty state.",
          onComposerChange,
        })}
      />,
    );

    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      emitFinalPhrases(["Confirm the copy", "and spacing"]);
    });

    expect(onComposerChange).toHaveBeenNthCalledWith(1, "Need the empty state. Confirm the copy");
    expect(onComposerChange).toHaveBeenNthCalledWith(2, "Need the empty state. Confirm the copy and spacing");
  });

  it("stops dictation before send", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    const onSend = vi.fn();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        analysis={analysisChat({
          view: WAITING_COMPOSER_VIEW,
          composer: "Need the empty state.",
          onSend,
        })}
      />,
    );

    await user.click(screen.getByTestId("dictate-button"));
    const recognition = latestRecognition();
    await user.click(screen.getByTestId("split-run-intent-composer-send"));

    expect(recognition.abort).toHaveBeenCalled();
    expect(onSend).toHaveBeenCalled();
  });

  it("shows a permission error toast and returns to idle", async () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const user = userEvent.setup();
    renderIntentDocument(
      <WorkOrderIntentDocument
        {...INTENT_DOC}
        artifacts={[INTENT]}
        analysis={analysisChat({ view: WAITING_COMPOSER_VIEW })}
      />,
    );

    await user.click(screen.getByTestId("dictate-button"));
    act(() => {
      latestRecognition().onerror?.({ error: "not-allowed" });
    });

    expect(showErrorToast).toHaveBeenCalledWith(ANALYSIS_PLANNING_COPY.microphoneDenied);
    expect(screen.getByTestId("dictate-button")).toHaveAttribute("aria-pressed", "false");
  });
});

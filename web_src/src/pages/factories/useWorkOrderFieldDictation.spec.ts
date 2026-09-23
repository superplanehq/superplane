import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { useWorkOrderFieldDictation } from "./useWorkOrderFieldDictation";

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

describe("useWorkOrderFieldDictation", () => {
  beforeEach(() => {
    FakeSpeechRecognition.instances = [];
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("keeps every final phrase from one recognition event", () => {
    const onTitleChange = vi.fn();
    const onDescriptionChange = vi.fn();
    const { result } = renderHook(() =>
      useWorkOrderFieldDictation({
        title: "Fix",
        description: "Refunds fail.",
        maxTitleLength: 256,
        maxDescriptionLength: 5000,
        onTitleChange,
        onDescriptionChange,
      }),
    );

    act(() => {
      result.current.rememberTitle();
      result.current.start();
      emitFinalPhrases(["refunds", "on retry"]);
    });

    expect(onTitleChange).toHaveBeenNthCalledWith(1, "Fix refunds");
    expect(onTitleChange).toHaveBeenNthCalledWith(2, "Fix refunds on retry");
    expect(onDescriptionChange).not.toHaveBeenCalled();
  });

  it("keeps every final phrase in the description field", () => {
    const onTitleChange = vi.fn();
    const onDescriptionChange = vi.fn();
    const { result } = renderHook(() =>
      useWorkOrderFieldDictation({
        title: "",
        description: "Refunds fail.",
        maxTitleLength: 256,
        maxDescriptionLength: 5000,
        onTitleChange,
        onDescriptionChange,
      }),
    );

    act(() => {
      result.current.start();
      emitFinalPhrases(["Retry the job", "Check logs"]);
    });

    expect(onDescriptionChange).toHaveBeenNthCalledWith(1, "Refunds fail. Retry the job");
    expect(onDescriptionChange).toHaveBeenNthCalledWith(2, "Refunds fail. Retry the job Check logs");
    expect(onTitleChange).not.toHaveBeenCalled();
  });

  it("writes the live phrase into the description and replaces later live words", () => {
    const onTitleChange = vi.fn();
    const onDescriptionChange = vi.fn();
    const { result } = renderHook(() =>
      useWorkOrderFieldDictation({
        title: "",
        description: "Refunds fail.",
        maxTitleLength: 256,
        maxDescriptionLength: 5000,
        onTitleChange,
        onDescriptionChange,
      }),
    );

    act(() => {
      result.current.start();
      emitTranscript("Retry", false);
      emitTranscript("Retry the job", false);
    });

    expect(onDescriptionChange).toHaveBeenLastCalledWith("Refunds fail. Retry the job");
    expect(onDescriptionChange.mock.calls.some((call) => call[0] === "Refunds fail. Retry Retry the job")).toBe(false);
    expect(onTitleChange).not.toHaveBeenCalled();
  });

  it("keeps in-progress words when stop is called", () => {
    const onDescriptionChange = vi.fn();
    const { result } = renderHook(() =>
      useWorkOrderFieldDictation({
        title: "",
        description: "",
        maxTitleLength: 256,
        maxDescriptionLength: 5000,
        onTitleChange: vi.fn(),
        onDescriptionChange,
      }),
    );

    act(() => {
      result.current.start();
      emitTranscript("Fix refunds", false);
      result.current.stop();
    });

    expect(onDescriptionChange).toHaveBeenCalledWith("Fix refunds");
    expect(onDescriptionChange).toHaveBeenLastCalledWith("Fix refunds");
  });

  it("treats a typed prefix as the snapshot when the live phrase is still a suffix", () => {
    const onDescriptionChange = vi.fn();
    const { result, rerender } = renderHook(
      ({ description }) =>
        useWorkOrderFieldDictation({
          title: "",
          description,
          maxTitleLength: 256,
          maxDescriptionLength: 5000,
          onTitleChange: vi.fn(),
          onDescriptionChange,
        }),
      { initialProps: { description: "" } },
    );

    act(() => {
      result.current.start();
      emitTranscript("Fix refunds", false);
    });
    rerender({ description: "Please Fix refunds" });
    act(() => {
      emitTranscript("Fix refunds on retry", false);
    });

    expect(onDescriptionChange).toHaveBeenLastCalledWith("Please Fix refunds on retry");
  });

  it("does not overwrite the title when focus moves during description dictation", () => {
    const onTitleChange = vi.fn();
    const onDescriptionChange = vi.fn();
    const { result } = renderHook(() =>
      useWorkOrderFieldDictation({
        title: "Fix bugs",
        description: "Refunds fail.",
        maxTitleLength: 256,
        maxDescriptionLength: 5000,
        onTitleChange,
        onDescriptionChange,
      }),
    );

    act(() => {
      result.current.start();
      emitTranscript("Retry", false);
      result.current.rememberTitle();
      emitTranscript("on checkout", false);
    });

    expect(onDescriptionChange).toHaveBeenLastCalledWith("Refunds fail. Retry");
    expect(onTitleChange).toHaveBeenLastCalledWith("Fix bugs on checkout");
    expect(onTitleChange.mock.calls.some((call) => String(call[0]).startsWith("Refunds fail."))).toBe(false);
  });

  it("replaces live words after focus returns before the phrase is final", () => {
    const onTitleChange = vi.fn();
    const onDescriptionChange = vi.fn();
    const { result, rerender } = renderHook(
      ({ description }) =>
        useWorkOrderFieldDictation({
          title: "Fix bugs",
          description,
          maxTitleLength: 256,
          maxDescriptionLength: 5000,
          onTitleChange,
          onDescriptionChange,
        }),
      { initialProps: { description: "Notes" } },
    );

    act(() => {
      result.current.start();
      emitTranscript("hello", false);
    });
    rerender({ description: "Notes hello" });
    act(() => {
      result.current.rememberTitle();
      result.current.rememberDescription();
      emitTranscript("hello", true);
    });

    expect(onDescriptionChange).toHaveBeenLastCalledWith("Notes hello");
    expect(onDescriptionChange.mock.calls.some((call) => call[0] === "Notes hello hello")).toBe(false);
    expect(onTitleChange).not.toHaveBeenCalled();
  });

  it("replaces live words after a newline when the field re-renders", () => {
    const onDescriptionChange = vi.fn();
    const { result, rerender } = renderHook(
      ({ description }) =>
        useWorkOrderFieldDictation({
          title: "",
          description,
          maxTitleLength: 256,
          maxDescriptionLength: 5000,
          onTitleChange: vi.fn(),
          onDescriptionChange,
        }),
      { initialProps: { description: "Notes\n" } },
    );

    act(() => {
      result.current.start();
      emitTranscript("hello", false);
    });
    rerender({ description: "Notes\nhello" });
    act(() => {
      emitTranscript("hello there", false);
    });

    expect(onDescriptionChange).toHaveBeenLastCalledWith("Notes\nhello there");
    expect(onDescriptionChange.mock.calls.some((call) => call[0] === "Notes\nhello hello there")).toBe(false);
  });

  it("replaces clipped live words when the final phrase changes", () => {
    const title = "A".repeat(250);
    const onTitleChange = vi.fn();
    const { result, rerender } = renderHook(
      ({ currentTitle }) =>
        useWorkOrderFieldDictation({
          title: currentTitle,
          description: "",
          maxTitleLength: 256,
          maxDescriptionLength: 5000,
          onTitleChange,
          onDescriptionChange: vi.fn(),
        }),
      { initialProps: { currentTitle: title } },
    );

    act(() => {
      result.current.rememberTitle();
      result.current.start();
      emitTranscript("refunds", false);
    });
    const clippedTitle = `${title} refun`;
    expect(onTitleChange).toHaveBeenLastCalledWith(clippedTitle);
    rerender({ currentTitle: clippedTitle });
    act(() => {
      emitTranscript("returns", true);
    });

    expect(onTitleChange).toHaveBeenLastCalledWith(`${title} retur`);
  });
});

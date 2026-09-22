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
});

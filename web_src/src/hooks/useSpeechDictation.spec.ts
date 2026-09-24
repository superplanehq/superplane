import { act, renderHook } from "@testing-library/react";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { useSpeechDictation } from "./useSpeechDictation";

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

function emitResult(transcript: string, isFinal: boolean) {
  latestRecognition().onresult?.({
    resultIndex: 0,
    results: [Object.assign([{ transcript }], { isFinal, 0: { transcript } })],
  });
}

describe("useSpeechDictation", () => {
  beforeEach(() => {
    FakeSpeechRecognition.instances = [];
  });

  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("is not supported when the browser has no speech recognition API", () => {
    const { result } = renderHook(() => useSpeechDictation({ onFinalPhrase: vi.fn() }));

    expect(result.current.isSupported).toBe(false);
    expect(result.current.isListening).toBe(false);
  });

  it("uses webkitSpeechRecognition when SpeechRecognition is missing", () => {
    vi.stubGlobal("webkitSpeechRecognition", FakeSpeechRecognition);
    const { result } = renderHook(() => useSpeechDictation({ onFinalPhrase: vi.fn() }));

    expect(result.current.isSupported).toBe(true);

    act(() => {
      result.current.start();
    });

    expect(latestRecognition().start).toHaveBeenCalledTimes(1);
  });

  it("does not start listening until start is called", () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    renderHook(() => useSpeechDictation({ onFinalPhrase: vi.fn() }));

    expect(FakeSpeechRecognition.instances).toHaveLength(0);
  });

  it("starts continuous listening with interim results and the browser language", () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    vi.stubGlobal("navigator", { language: "fr-FR" });
    const { result } = renderHook(() => useSpeechDictation({ onFinalPhrase: vi.fn() }));

    act(() => {
      result.current.start();
    });

    const recognition = latestRecognition();
    expect(result.current.isListening).toBe(true);
    expect(recognition.continuous).toBe(true);
    expect(recognition.interimResults).toBe(true);
    expect(recognition.lang).toBe("fr-FR");
    expect(recognition.start).toHaveBeenCalledTimes(1);
  });

  it("exposes the interim phrase and does not append it", () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const onFinalPhrase = vi.fn();
    const { result } = renderHook(() => useSpeechDictation({ onFinalPhrase }));

    act(() => {
      result.current.start();
      emitResult("Fix refunds", false);
    });

    expect(result.current.interimPhrase).toBe("Fix refunds");
    expect(onFinalPhrase).not.toHaveBeenCalled();
  });

  it("appends a final phrase once and clears the interim phrase", () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const onFinalPhrase = vi.fn();
    const { result } = renderHook(() => useSpeechDictation({ onFinalPhrase }));

    act(() => {
      result.current.start();
      emitResult("Fix refunds", false);
      emitResult("Fix refunds", true);
    });

    expect(onFinalPhrase).toHaveBeenCalledTimes(1);
    expect(onFinalPhrase).toHaveBeenCalledWith("Fix refunds");
    expect(result.current.interimPhrase).toBe("");
  });

  it("returns to idle and reports a permission error for not-allowed", () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const { result } = renderHook(() => useSpeechDictation({ onFinalPhrase: vi.fn() }));

    act(() => {
      result.current.start();
      latestRecognition().onerror?.({ error: "not-allowed" });
    });

    expect(result.current.isListening).toBe(false);
    expect(result.current.permissionError).toBe(true);
    expect(result.current.interimPhrase).toBe("");
  });

  it("returns to idle with no permission error for other listen errors", () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const { result } = renderHook(() => useSpeechDictation({ onFinalPhrase: vi.fn() }));

    act(() => {
      result.current.start();
      latestRecognition().onerror?.({ error: "network" });
    });

    expect(result.current.isListening).toBe(false);
    expect(result.current.permissionError).toBe(false);
  });

  it("stops listening when stop is called", () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const { result } = renderHook(() => useSpeechDictation({ onFinalPhrase: vi.fn() }));

    act(() => {
      result.current.start();
    });
    const recognition = latestRecognition();
    act(() => {
      result.current.stop();
    });

    expect(recognition.abort).toHaveBeenCalledTimes(1);
    expect(result.current.isListening).toBe(false);
  });

  it("aborts recognition on unmount", () => {
    vi.stubGlobal("SpeechRecognition", FakeSpeechRecognition);
    const { result, unmount } = renderHook(() => useSpeechDictation({ onFinalPhrase: vi.fn() }));

    act(() => {
      result.current.start();
    });
    const recognition = latestRecognition();
    unmount();

    expect(recognition.abort).toHaveBeenCalledTimes(1);
  });
});

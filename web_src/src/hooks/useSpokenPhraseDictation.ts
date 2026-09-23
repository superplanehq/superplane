import { useRef, type MutableRefObject } from "react";

import { appendSpokenPhrase, stripTrailingSpokenPhrase } from "@/lib/appendSpokenPhrase";

import { useSpeechDictation, type UseSpeechDictationResult } from "./useSpeechDictation";

export type SpokenPhraseField = {
  key?: string;
  getValue: () => string;
  setValue: (next: string) => void;
  maxLength?: number;
};

type FieldSnapshot = {
  committed: string;
  livePhrase: string;
  segmentId: number;
};

export type UseSpokenPhraseDictationResult = UseSpeechDictationResult & {
  activateField: (nextKey: string) => void;
};

export function useSpokenPhraseDictation(
  fieldRef: MutableRefObject<SpokenPhraseField>,
): UseSpokenPhraseDictationResult {
  const snapshotsRef = useRef(new Map<string, FieldSnapshot>());
  const fieldKeyRef = useRef(fieldRef.current.key ?? "default");
  const committedRef = useRef(fieldRef.current.getValue());
  const livePhraseRef = useRef("");
  const valueRef = useRef(fieldRef.current.getValue());
  const segmentIdRef = useRef(0);

  const write = (next: string) => {
    if (next === valueRef.current) {
      return;
    }
    valueRef.current = next;
    fieldRef.current.setValue(next);
  };

  const syncFromField = () => {
    const currentValue = fieldRef.current.getValue();
    valueRef.current = currentValue;
    const expectedValue = appendSpokenPhrase(committedRef.current, livePhraseRef.current, fieldRef.current.maxLength);
    if (currentValue !== expectedValue) {
      committedRef.current = stripTrailingSpokenPhrase(currentValue, livePhraseRef.current);
    }
  };

  const resetSnapshot = () => {
    snapshotsRef.current.clear();
    livePhraseRef.current = "";
    committedRef.current = fieldRef.current.getValue();
    valueRef.current = committedRef.current;
  };

  const finishSegment = () => {
    segmentIdRef.current += 1;
  };

  const restoreSnapshot = (saved: FieldSnapshot) => {
    if (saved.livePhrase && saved.segmentId !== segmentIdRef.current) {
      committedRef.current = appendSpokenPhrase(saved.committed, saved.livePhrase);
      livePhraseRef.current = "";
      return;
    }
    committedRef.current = saved.committed;
    livePhraseRef.current = saved.livePhrase;
  };

  const activateField = (nextKey: string) => {
    if (nextKey === fieldKeyRef.current) {
      return;
    }
    snapshotsRef.current.set(fieldKeyRef.current, {
      committed: committedRef.current,
      livePhrase: livePhraseRef.current,
      segmentId: segmentIdRef.current,
    });
    fieldKeyRef.current = nextKey;
    const saved = snapshotsRef.current.get(nextKey);
    if (saved) {
      restoreSnapshot(saved);
    } else {
      livePhraseRef.current = "";
      committedRef.current = fieldRef.current.getValue();
    }
    syncFromField();
  };

  syncFromField();

  const dictation = useSpeechDictation({
    onFinalPhrase: (phrase) => {
      const next = appendSpokenPhrase(committedRef.current, phrase, fieldRef.current.maxLength);
      committedRef.current = next;
      livePhraseRef.current = "";
      write(next);
      finishSegment();
    },
    onInterimPhrase: (livePhrase) => {
      livePhraseRef.current = livePhrase;
      write(appendSpokenPhrase(committedRef.current, livePhrase, fieldRef.current.maxLength));
    },
  });

  return {
    ...dictation,
    activateField,
    start: () => {
      resetSnapshot();
      dictation.start();
    },
    stop: () => {
      const livePhrase = livePhraseRef.current || dictation.interimPhrase;
      if (livePhrase.trim()) {
        const next = appendSpokenPhrase(committedRef.current, livePhrase, fieldRef.current.maxLength);
        committedRef.current = next;
        livePhraseRef.current = "";
        write(next);
      }
      finishSegment();
      dictation.stop();
    },
  };
}

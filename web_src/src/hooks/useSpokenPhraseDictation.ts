import { useRef, type MutableRefObject } from "react";

import { appendSpokenPhrase, stripLeadingSpokenPhrase, stripTrailingSpokenPhrase } from "@/lib/appendSpokenPhrase";

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
};

export type UseSpokenPhraseDictationResult = UseSpeechDictationResult & {
  activateField: (nextKey: string) => void;
};

function snapshotAfterFinalPhrase(saved: FieldSnapshot, phrase: string): FieldSnapshot {
  if (!saved.livePhrase) {
    return saved;
  }
  const remaining = stripLeadingSpokenPhrase(saved.livePhrase, phrase);
  if (remaining === null) {
    return {
      committed: appendSpokenPhrase(saved.committed, saved.livePhrase),
      livePhrase: "",
    };
  }
  return {
    committed: appendSpokenPhrase(saved.committed, phrase),
    livePhrase: remaining,
  };
}

export function useSpokenPhraseDictation(
  fieldRef: MutableRefObject<SpokenPhraseField>,
): UseSpokenPhraseDictationResult {
  const snapshotsRef = useRef(new Map<string, FieldSnapshot>());
  const fieldKeyRef = useRef(fieldRef.current.key ?? "default");
  const committedRef = useRef(fieldRef.current.getValue());
  const livePhraseRef = useRef("");
  const valueRef = useRef(fieldRef.current.getValue());

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

  const applyFinalPhraseToSnapshots = (phrase: string) => {
    const snapshots = snapshotsRef.current;
    for (const [key, saved] of snapshots) {
      if (key === fieldKeyRef.current) {
        continue;
      }
      snapshots.set(key, snapshotAfterFinalPhrase(saved, phrase));
    }
  };

  const restoreSnapshot = (saved: FieldSnapshot) => {
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
      applyFinalPhraseToSnapshots(phrase);
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
      dictation.stop();
    },
  };
}

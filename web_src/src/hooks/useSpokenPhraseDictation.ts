import { useRef, type MutableRefObject } from "react";

import { appendSpokenPhrase, stripTrailingSpokenPhrase } from "@/lib/appendSpokenPhrase";

import { useSpeechDictation, type UseSpeechDictationResult } from "./useSpeechDictation";

export type SpokenPhraseField = {
  getValue: () => string;
  setValue: (next: string) => void;
  maxLength?: number;
};

export type UseSpokenPhraseDictationResult = UseSpeechDictationResult & {
  resetSnapshot: () => void;
};

export function useSpokenPhraseDictation(
  fieldRef: MutableRefObject<SpokenPhraseField>,
): UseSpokenPhraseDictationResult {
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

  const resetSnapshot = () => {
    livePhraseRef.current = "";
    committedRef.current = fieldRef.current.getValue();
    valueRef.current = committedRef.current;
  };

  const currentValue = fieldRef.current.getValue();
  valueRef.current = currentValue;
  const expectedValue = appendSpokenPhrase(committedRef.current, livePhraseRef.current, fieldRef.current.maxLength);
  if (currentValue !== expectedValue) {
    committedRef.current = stripTrailingSpokenPhrase(currentValue, livePhraseRef.current);
  }

  const dictation = useSpeechDictation({
    onFinalPhrase: (phrase) => {
      const next = appendSpokenPhrase(committedRef.current, phrase, fieldRef.current.maxLength);
      committedRef.current = next;
      livePhraseRef.current = "";
      write(next);
    },
    onInterimPhrase: (livePhrase) => {
      livePhraseRef.current = livePhrase;
      write(appendSpokenPhrase(committedRef.current, livePhrase, fieldRef.current.maxLength));
    },
  });

  return {
    ...dictation,
    resetSnapshot,
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

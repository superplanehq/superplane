import { useRef, type MutableRefObject } from "react";

import { appendSpokenPhrase, stripTrailingSpokenPhrase } from "@/lib/appendSpokenPhrase";

import { useSpeechDictation, type UseSpeechDictationResult } from "./useSpeechDictation";

export type SpokenPhraseField = {
  getValue: () => string;
  setValue: (next: string) => void;
  maxLength?: number;
};

export function useSpokenPhraseDictation(fieldRef: MutableRefObject<SpokenPhraseField>): UseSpeechDictationResult {
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

  const currentValue = fieldRef.current.getValue();
  valueRef.current = currentValue;
  committedRef.current = stripTrailingSpokenPhrase(currentValue, livePhraseRef.current);

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
    start: () => {
      committedRef.current = fieldRef.current.getValue();
      livePhraseRef.current = "";
      valueRef.current = committedRef.current;
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

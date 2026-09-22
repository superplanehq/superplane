import { useRef } from "react";

import { useSpeechDictation, type UseSpeechDictationResult } from "@/hooks/useSpeechDictation";
import { appendSpokenPhrase } from "@/lib/appendSpokenPhrase";

export type WorkOrderDictationField = "title" | "description";

export function useWorkOrderFieldDictation({
  title,
  description,
  maxTitleLength,
  maxDescriptionLength,
  onTitleChange,
  onDescriptionChange,
}: {
  title: string;
  description: string;
  maxTitleLength: number;
  maxDescriptionLength: number;
  onTitleChange: (next: string) => void;
  onDescriptionChange: (next: string) => void;
}): UseSpeechDictationResult & {
  rememberTitle: () => void;
  rememberDescription: () => void;
} {
  const lastFieldRef = useRef<WorkOrderDictationField>("description");
  const dictation = useSpeechDictation({
    onFinalPhrase: (phrase) => {
      if (lastFieldRef.current === "title") {
        onTitleChange(appendSpokenPhrase(title, phrase, maxTitleLength));
        return;
      }
      onDescriptionChange(appendSpokenPhrase(description, phrase, maxDescriptionLength));
    },
  });

  return {
    ...dictation,
    rememberTitle: () => {
      lastFieldRef.current = "title";
    },
    rememberDescription: () => {
      lastFieldRef.current = "description";
    },
  };
}

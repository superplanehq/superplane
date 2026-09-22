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
  const titleRef = useRef(title);
  const descriptionRef = useRef(description);
  titleRef.current = title;
  descriptionRef.current = description;
  const dictation = useSpeechDictation({
    onFinalPhrase: (phrase) => {
      if (lastFieldRef.current === "title") {
        const next = appendSpokenPhrase(titleRef.current, phrase, maxTitleLength);
        titleRef.current = next;
        onTitleChange(next);
        return;
      }
      const next = appendSpokenPhrase(descriptionRef.current, phrase, maxDescriptionLength);
      descriptionRef.current = next;
      onDescriptionChange(next);
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

import { useCallback, useEffect, useRef, useState } from "react";

type SpeechRecognitionAlternativeLike = {
  transcript: string;
};

type SpeechRecognitionResultLike = {
  isFinal: boolean;
  0?: SpeechRecognitionAlternativeLike;
};

type SpeechRecognitionEventLike = {
  resultIndex: number;
  results: ArrayLike<SpeechRecognitionResultLike>;
};

type SpeechRecognitionErrorLike = {
  error: string;
};

type SpeechRecognitionLike = {
  continuous: boolean;
  interimResults: boolean;
  lang: string;
  start: () => void;
  stop: () => void;
  abort: () => void;
  onresult: ((event: SpeechRecognitionEventLike) => void) | null;
  onerror: ((event: SpeechRecognitionErrorLike) => void) | null;
  onend: (() => void) | null;
};

type SpeechRecognitionConstructor = new () => SpeechRecognitionLike;

type SpeechWindow = Window & {
  SpeechRecognition?: SpeechRecognitionConstructor;
  webkitSpeechRecognition?: SpeechRecognitionConstructor;
};

export type UseSpeechDictationArgs = {
  onFinalPhrase: (phrase: string) => void;
  onInterimPhrase?: (phrase: string) => void;
};

export type UseSpeechDictationResult = {
  isSupported: boolean;
  isListening: boolean;
  interimPhrase: string;
  permissionError: boolean;
  start: () => void;
  stop: () => void;
};

function speechRecognitionConstructor(): SpeechRecognitionConstructor | undefined {
  if (typeof window === "undefined") {
    return undefined;
  }
  const speechWindow = window as SpeechWindow;
  return speechWindow.SpeechRecognition ?? speechWindow.webkitSpeechRecognition;
}

function isSpeechDictationSupported(): boolean {
  return Boolean(speechRecognitionConstructor());
}

export function useSpeechDictation({
  onFinalPhrase,
  onInterimPhrase,
}: UseSpeechDictationArgs): UseSpeechDictationResult {
  const [isListening, setIsListening] = useState(false);
  const [interimPhrase, setInterimPhrase] = useState("");
  const [permissionError, setPermissionError] = useState(false);
  const onFinalPhraseRef = useRef(onFinalPhrase);
  const onInterimPhraseRef = useRef(onInterimPhrase);
  const recognitionRef = useRef<SpeechRecognitionLike | null>(null);
  const listeningWantedRef = useRef(false);
  const mountedRef = useRef(true);

  onFinalPhraseRef.current = onFinalPhrase;
  onInterimPhraseRef.current = onInterimPhrase;

  const stopRecognition = useCallback(() => {
    listeningWantedRef.current = false;
    const recognition = recognitionRef.current;
    recognitionRef.current = null;
    recognition?.abort();
    setIsListening(false);
    setInterimPhrase("");
  }, []);

  useEffect(() => {
    mountedRef.current = true;
    return () => {
      mountedRef.current = false;
      listeningWantedRef.current = false;
      recognitionRef.current?.abort();
      recognitionRef.current = null;
    };
  }, []);

  const start = useCallback(() => {
    const Recognition = speechRecognitionConstructor();
    if (!Recognition || listeningWantedRef.current) {
      return;
    }

    setPermissionError(false);
    const recognition = new Recognition();
    recognition.continuous = true;
    recognition.interimResults = true;
    recognition.lang = typeof navigator === "undefined" ? "en-US" : navigator.language;
    recognition.onresult = (event) => {
      if (!mountedRef.current || !listeningWantedRef.current) {
        return;
      }
      let interim = "";
      for (let index = event.resultIndex; index < event.results.length; index += 1) {
        const result = event.results[index];
        const transcript = result?.[0]?.transcript ?? "";
        if (result?.isFinal) {
          const spoken = transcript.trim();
          if (spoken) {
            onFinalPhraseRef.current(spoken);
          }
        } else {
          interim += transcript;
        }
      }
      setInterimPhrase(interim);
      onInterimPhraseRef.current?.(interim);
    };
    recognition.onerror = (event) => {
      listeningWantedRef.current = false;
      recognitionRef.current = null;
      if (!mountedRef.current) {
        return;
      }
      setIsListening(false);
      setInterimPhrase("");
      if (event.error === "not-allowed") {
        setPermissionError(true);
      }
    };
    recognition.onend = () => {
      if (!mountedRef.current || !listeningWantedRef.current) {
        return;
      }
      try {
        recognition.start();
      } catch {
        listeningWantedRef.current = false;
        recognitionRef.current = null;
        setIsListening(false);
        setInterimPhrase("");
      }
    };

    recognitionRef.current = recognition;
    listeningWantedRef.current = true;
    setIsListening(true);
    setInterimPhrase("");
    try {
      recognition.start();
    } catch {
      stopRecognition();
    }
  }, [stopRecognition]);

  return {
    isSupported: isSpeechDictationSupported(),
    isListening,
    interimPhrase,
    permissionError,
    start,
    stop: stopRecognition,
  };
}

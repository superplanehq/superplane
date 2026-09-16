import { useEffect, useRef, useState } from "react";

const STATUS_SWAP_MS = 180;

type StatusFrame = {
  current: string;
  outgoing?: string;
};

export function AnimatedThinkingState({ text }: { text: string }) {
  const latestText = useRef(text);
  const [frame, setFrame] = useState<StatusFrame>({ current: text });

  useEffect(() => {
    if (text === latestText.current) return;

    const outgoing = latestText.current;
    latestText.current = text;
    setFrame({ current: text, outgoing });

    const timer = window.setTimeout(() => {
      setFrame((current) => (current.current === text ? { current: text } : current));
    }, STATUS_SWAP_MS);
    return () => window.clearTimeout(timer);
  }, [text]);

  const sizerText = longerText(frame.current, frame.outgoing);
  return (
    <span className="sp-thinking-state" aria-hidden>
      <span className="sp-thinking-state-sizer" data-text={sizerText} />
      {frame.outgoing ? (
        <span key={frame.outgoing} className="sp-ai-thinking sp-thinking-state-outgoing" data-text={frame.outgoing}>
          {frame.outgoing}
        </span>
      ) : null}
      <span key={frame.current} className="sp-ai-thinking sp-thinking-state-current" data-text={frame.current}>
        {frame.current}
      </span>
    </span>
  );
}

function longerText(current: string, outgoing?: string): string {
  return outgoing && outgoing.length > current.length ? outgoing : current;
}

import mermaid from "mermaid";
import { useEffect, useId, useRef, useState } from "react";

mermaid.initialize({
  startOnLoad: false,
  theme: "neutral",
  flowchart: { curve: "basis", padding: 12 },
  securityLevel: "loose",
});

export type MermaidDiagramProps = {
  definition: string;
  className?: string;
};

export function MermaidDiagram({ definition, className }: MermaidDiagramProps) {
  const containerRef = useRef<HTMLDivElement>(null);
  const uniqueId = useId().replace(/:/g, "_");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    const container = containerRef.current;
    if (!container || !definition.trim()) {
      return;
    }

    let cancelled = false;

    void (async () => {
      try {
        const { svg } = await mermaid.render(`mermaid-${uniqueId}`, definition);
        if (!cancelled && container) {
          container.innerHTML = svg;
          setError(null);
        }
      } catch (err) {
        if (!cancelled) {
          setError(err instanceof Error ? err.message : "Failed to render diagram");
          container.innerHTML = "";
        }
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [definition, uniqueId]);

  if (error) {
    return <div className="text-xs text-muted-foreground italic px-2 py-3">Could not render diagram preview.</div>;
  }

  return <div ref={containerRef} className={className} />;
}

import React, { useEffect, useRef, useState } from "react";

/**
 * Fabrica PoC for superplane#5368 — Markdown view: render ```mermaid blocks as
 * diagrams and @node references as clickable chips.
 *
 * Self-contained so it can be dropped into MarkdownContent (web_src/src/pages/app/Markdown.tsx)
 * alongside the existing MermaidWidget / NodeChip components. Mermaid is lazy-imported
 * so it only loads when a diagram is actually present (keeps bundle impact isolated).
 */
type Props = { source: string; onNodeClick?: (nodeId: string) => void };

export function MermaidMarkdown({ source, onNodeClick }: Props) {
  const segments = source.split(/```mermaid\n([\s\S]*?)```/g);
  return (
    <div className="fabrica-markdown">
      {segments.map((seg, i) =>
        i % 2 === 1 ? (
          <Mermaid key={i} code={seg} />
        ) : (
          <Text key={i} text={seg} onNodeClick={onNodeClick} />
        )
      )}
    </div>
  );
}

function Mermaid({ code }: { code: string }) {
  const ref = useRef<HTMLDivElement>(null);
  const [err, setErr] = useState(false);
  useEffect(() => {
    let alive = true;
    import("mermaid")
      .then(async (m) => {
        const id = "mmd-" + Math.random().toString(36).slice(2);
        const { svg } = await m.default.render(id, code.trim());
        if (alive && ref.current) ref.current.innerHTML = svg;
      })
      .catch(() => alive && setErr(true));
    return () => {
      alive = false;
    };
  }, [code]);
  if (err) return <pre className="mermaid-error">⚠ Diagram error\n{code}</pre>;
  return <div ref={ref} className="mermaid" />;
}

function Text({ text, onNodeClick }: { text: string; onNodeClick?: (id: string) => void }) {
  const parts = text.split(/(@[A-Za-z0-9_-]+)/g);
  return (
    <span>
      {parts.map((p, i) =>
        p.startsWith("@") ? (
          <button
            key={i}
            className="node-chip"
            onClick={() => onNodeClick?.(p.slice(1))}
          >
            {p}
          </button>
        ) : (
          <React.Fragment key={i}>{p}</React.Fragment>
        )
      )}
    </span>
  );
}

export default MermaidMarkdown;

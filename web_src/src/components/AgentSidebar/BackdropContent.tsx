/**
 * Renders the text with @mentions highlighted behind a transparent textarea.
 */
export function BackdropContent({
  text,
  mentions,
}: {
  text: string;
  mentions: { label: string; startIndex: number }[];
}) {
  if (mentions.length === 0) {
    // No mentions — render the text normally to maintain layout
    return <span className="whitespace-pre-wrap break-words text-[rgba(10,10,10,1)]">{text || "\u00A0"}</span>;
  }

  // Build segments using tracked startIndex positions for accurate rendering
  const sorted = [...mentions]
    .filter((m) => {
      const expected = `@${m.label}`;
      return text.slice(m.startIndex, m.startIndex + expected.length) === expected;
    })
    .sort((a, b) => a.startIndex - b.startIndex);

  const segments: { text: string; isMention: boolean }[] = [];
  let pos = 0;

  for (const m of sorted) {
    const displayText = `@${m.label}`;
    if (m.startIndex > pos) {
      segments.push({ text: text.slice(pos, m.startIndex), isMention: false });
    }
    segments.push({ text: displayText, isMention: true });
    pos = m.startIndex + displayText.length;
  }

  if (pos < text.length) {
    segments.push({ text: text.slice(pos), isMention: false });
  }

  return (
    <>
      {segments.map((seg, i) =>
        seg.isMention ? (
          <span key={i} className="rounded bg-blue-100 text-blue-700">
            {seg.text}
          </span>
        ) : (
          <span key={i} className="whitespace-pre-wrap text-[rgba(10,10,10,1)]">
            {seg.text}
          </span>
        ),
      )}
    </>
  );
}

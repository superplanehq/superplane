import { useRef, useState } from "react";

export function useEditorCommittedContent() {
  const [committedContentByPath, setCommittedContentByPath] = useState<Record<string, string>>({});
  const committedContentByPathRef = useRef(committedContentByPath);
  committedContentByPathRef.current = committedContentByPath;

  return { committedContentByPath, setCommittedContentByPath, committedContentByPathRef };
}

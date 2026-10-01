import { useState } from "react";

import { cn } from "@/lib/utils";

export function GitHubAttachmentMedia({ src, alt, className }: { src: string; alt?: string; className?: string }) {
  const [mode, setMode] = useState<"video" | "image" | "link">("video");
  const label = alt?.trim() || "Attachment";
  const mediaClassName = cn("max-w-full", className);

  if (mode === "video") {
    return (
      <video
        src={src}
        className={mediaClassName}
        controls
        playsInline
        aria-label={label}
        onError={() => setMode("image")}
      />
    );
  }

  if (mode === "image") {
    return <img src={src} alt={label} className={mediaClassName} onError={() => setMode("link")} />;
  }

  return (
    <a href={src} target="_blank" rel="noopener noreferrer">
      {src}
    </a>
  );
}

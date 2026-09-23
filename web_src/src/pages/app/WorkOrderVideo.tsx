import { useEffect, useState } from "react";

import { isBrowserPlayableWorkOrderVideo } from "@/lib/workOrderFiles";

export function WorkOrderVideo({
  src,
  alt,
  className,
  contentType,
}: {
  src: string;
  alt?: string;
  className?: string;
  contentType?: string;
}) {
  const [failed, setFailed] = useState(!isBrowserPlayableWorkOrderVideo(contentType, src, alt));
  const label = alt?.trim() || "Video";

  useEffect(() => {
    setFailed(!isBrowserPlayableWorkOrderVideo(contentType, src, alt));
  }, [alt, contentType, src]);

  if (failed) {
    return (
      <span className="work-order-video-fallback inline-flex max-w-full flex-col gap-1 rounded-md border border-border bg-muted/40 px-3 py-2 text-sm">
        <span className="font-medium">{label}</span>
        <a href={src} download className="text-sky-600 underline">
          Download video
        </a>
        <span>This browser cannot play this video.</span>
      </span>
    );
  }

  return (
    <video src={src} className={className} controls playsInline aria-label={label} onError={() => setFailed(true)} />
  );
}

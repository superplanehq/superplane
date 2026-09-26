import { useEffect, useState } from "react";

import {
  isBrowserPlayableWorkOrderAudio,
  isBrowserPlayableWorkOrderVideo,
  isWorkOrderAudioSource,
} from "@/lib/workOrderFiles";

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
  const audio = isWorkOrderAudioSource({ contentType, src, alt });
  const playable = audio
    ? isBrowserPlayableWorkOrderAudio(contentType, src, alt)
    : isBrowserPlayableWorkOrderVideo(contentType, src, alt);
  const [failed, setFailed] = useState(!playable);
  const label = alt?.trim() || (audio ? "Audio" : "Video");
  const kind = audio ? "audio" : "video";

  useEffect(() => {
    setFailed(!playable);
  }, [playable]);

  if (failed) {
    return (
      <span className="work-order-video-fallback inline-flex max-w-full flex-col gap-1 rounded-md border border-border bg-muted/40 px-3 py-2 text-sm">
        <span className="font-medium">{label}</span>
        <a href={src} download className="text-sky-600 underline">
          {audio ? "Download audio" : "Download video"}
        </a>
        <span>This browser cannot play this {kind}.</span>
      </span>
    );
  }

  if (audio) {
    return <audio src={src} className={className} controls aria-label={label} onError={() => setFailed(true)} />;
  }

  return (
    <video src={src} className={className} controls playsInline aria-label={label} onError={() => setFailed(true)} />
  );
}

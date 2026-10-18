import { HOSTED_VIDEO_COPY, type HostedVideo } from "@/lib/hostedVideo";
import { cn } from "@/lib/utils";

export function HostedVideoEmbed({ video, className }: { video: HostedVideo; className?: string }) {
  if (!video.embedUrl) {
    return (
      <span
        className="hosted-video-card mt-3 flex w-fit max-w-full flex-col gap-1 rounded-md border border-border bg-muted/40 px-3 py-2 text-sm"
        style={{ display: "flex", flexDirection: "column", gap: "0.25rem", marginTop: "0.75rem", width: "fit-content" }}
      >
        <span className="font-medium">{video.providerName}</span>
        <a href={video.pageUrl} target="_blank" rel="noopener noreferrer">
          {HOSTED_VIDEO_COPY.open}
        </a>
      </span>
    );
  }

  return (
    <span
      className={cn("hosted-video-frame block w-full max-w-xl", className)}
      style={{ display: "block", width: "100%", maxWidth: "36rem" }}
    >
      <iframe
        src={video.embedUrl}
        title={video.providerName}
        className="aspect-video w-full max-w-xl rounded-md border border-border"
        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
        allowFullScreen
      />
    </span>
  );
}

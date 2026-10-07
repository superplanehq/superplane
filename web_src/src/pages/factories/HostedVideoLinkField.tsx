import { Video } from "lucide-react";
import { useState, type KeyboardEvent } from "react";

import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { HOSTED_VIDEO_COPY, hostedVideoFieldError, parseHostedVideoUrl, type HostedVideo } from "@/lib/hostedVideo";

export function HostedVideoLinkField({
  disabled = false,
  onAdd,
}: {
  disabled?: boolean;
  onAdd: (video: HostedVideo) => boolean;
}) {
  const [open, setOpen] = useState(false);
  const [value, setValue] = useState("");
  const error = hostedVideoFieldError(value);

  const apply = () => {
    const video = parseHostedVideoUrl(value);
    if (!video) {
      return;
    }
    if (!onAdd(video)) {
      return;
    }
    setValue("");
    setOpen(false);
  };

  const onInputKeyDown = (event: KeyboardEvent<HTMLInputElement>) => {
    if (event.key !== "Enter") {
      return;
    }
    event.preventDefault();
    event.stopPropagation();
    apply();
  };

  return (
    <div className="flex min-w-0 flex-col gap-1">
      <div className="flex min-w-0 items-center gap-1">
        <Button
          type="button"
          variant="ghost"
          size="icon"
          className="size-8 rounded-full text-muted-foreground"
          disabled={disabled}
          aria-label={HOSTED_VIDEO_COPY.add}
          aria-expanded={open}
          data-testid="hosted-video-link-toggle"
          onClick={() => setOpen((current) => !current)}
        >
          <Video className="size-4" aria-hidden />
        </Button>
        {open ? (
          <div className="flex min-w-0 items-center gap-1">
            <Input
              aria-label={HOSTED_VIDEO_COPY.label}
              value={value}
              disabled={disabled}
              placeholder={HOSTED_VIDEO_COPY.placeholder}
              data-testid="hosted-video-link-input"
              className="h-8 w-44 text-[12px]"
              onChange={(event) => setValue(event.target.value)}
              onKeyDown={onInputKeyDown}
            />
            <Button
              type="button"
              size="sm"
              className="h-8 px-2 text-[12px]"
              disabled={disabled || !parseHostedVideoUrl(value)}
              onClick={apply}
            >
              {HOSTED_VIDEO_COPY.apply}
            </Button>
          </div>
        ) : null}
      </div>
      {open ? (
        <p className="max-w-64 text-[11px] text-muted-foreground" data-testid="hosted-video-link-helper">
          {error || HOSTED_VIDEO_COPY.helper}
        </p>
      ) : null}
    </div>
  );
}

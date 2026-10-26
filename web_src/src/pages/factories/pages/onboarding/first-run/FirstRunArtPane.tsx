import { cn } from "@/lib/utils";
import { useEffect, useRef } from "react";

import { artInk, type FirstRunArtInk, type FirstRunArtScene } from "./firstRunArtScene";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import {
  loadOnboardingArt,
  mountOnboardingArt,
  ONBOARDING_ART_COLOR_TWEEN_MS,
  type OnboardingArtHandle,
} from "./onboardingArt";

export function FirstRunArtPane({
  art,
  testId,
  className,
}: {
  art: FirstRunArtScene;
  testId?: string;
  className?: string;
}) {
  const canvasRef = useRef<HTMLDivElement>(null);
  const handleRef = useRef<OnboardingArtHandle | null>(null);
  const { mode, background, arrowColor, count, arrowScale } = art;

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    let cancelled = false;
    const scene: FirstRunArtScene = { mode, background, arrowColor, count, arrowScale };

    const mount = () => {
      if (cancelled) return;
      const handle = mountOnboardingArt(canvas, scene);
      if (handle) handleRef.current = handle;
    };

    const sync = async () => {
      try {
        await loadOnboardingArt();
      } catch {
        return;
      }
      if (cancelled) return;
      const current = handleRef.current;
      if (!current) {
        mount();
        return;
      }
      current.setColors(background, arrowColor, ONBOARDING_ART_COLOR_TWEEN_MS, () => {
        if (cancelled) return;
        current.destroy();
        if (handleRef.current === current) handleRef.current = null;
        mount();
      });
    };

    void sync();
    return () => {
      cancelled = true;
    };
  }, [mode, background, arrowColor, count, arrowScale]);

  useEffect(() => {
    return () => {
      handleRef.current?.destroy();
      handleRef.current = null;
    };
  }, []);

  return (
    <div
      className={cn("absolute inset-0", className)}
      data-testid={testId ?? "first-run-art-pane"}
      data-art-mode={art.mode}
      data-art-background={art.background}
      data-art-arrow-color={art.arrowColor}
      data-art-count={art.count ?? ""}
    >
      <div ref={canvasRef} className="absolute inset-0" data-testid="first-run-art-canvas" />
      {art.badges ? <ArtBadges discover={art.badges.discover} verify={art.badges.verify} /> : null}
      {!art.badges && art.pill ? <ArtPill pill={art.pill} ink={artInk(art)} /> : null}
    </div>
  );
}

function pillTone(pill: NonNullable<FirstRunArtScene["pill"]>, ink: FirstRunArtInk) {
  if (ink === "dark") {
    return { box: "bg-[rgba(17,17,14,0.08)] text-[#11110e]", label: "text-[rgba(17,17,14,0.65)]" };
  }
  if (pill.light) {
    return { box: "bg-[rgba(163,162,158,0.08)] text-[#a3a29e]", label: "text-[rgba(163,162,158,0.7)]" };
  }
  return { box: "bg-[rgba(255,255,255,0.08)] text-[#f9f9f3]", label: "text-[rgba(228,228,218,0.7)]" };
}

function ArtPill({ pill, ink }: { pill: NonNullable<FirstRunArtScene["pill"]>; ink: FirstRunArtInk }) {
  const tone = pillTone(pill, ink);
  return (
    <div
      className={cn(
        "absolute bottom-[9%] left-1/2 z-10 flex -translate-x-1/2 items-center gap-[7px] whitespace-nowrap rounded-[4px] px-3 py-2 font-mono text-[13px] uppercase tracking-[1.3px]",
        tone.box,
      )}
      data-testid="first-run-art-pill"
      data-ink={ink}
    >
      <span className={tone.label}>{pill.label}</span>
      <span aria-hidden>→</span>
      <span>{pill.value}</span>
    </div>
  );
}

function ArtBadges({ discover, verify }: { discover: string; verify: string }) {
  const copy = FIRST_RUN_COPY.sphere;
  return (
    <>
      <div
        className="absolute top-[24%] left-[11%] z-10 flex flex-col gap-[5px] rounded-lg border border-[#34322b] bg-[rgba(13,13,11,0.85)] px-3 py-3 font-mono text-[13px] whitespace-nowrap"
        data-testid="first-run-art-badge-discover"
      >
        <span className="text-[#d69717]">{copy.discover}</span>
        <span className="text-[#eeede9]">{discover}</span>
      </div>
      <div
        className="absolute right-[8%] bottom-[22%] z-10 flex flex-col gap-[5px] rounded-lg border border-[#d69717] bg-[rgba(13,13,11,0.85)] px-3 py-3 font-mono text-[13px] whitespace-nowrap text-[#d69717]"
        data-testid="first-run-art-badge-verify"
      >
        <span>{copy.verify}</span>
        <span>{verify}</span>
      </div>
    </>
  );
}

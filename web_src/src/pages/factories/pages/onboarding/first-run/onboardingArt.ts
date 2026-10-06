import type { FirstRunArtScene } from "./firstRunArtScene";

const COLOR_TWEEN_MS = 1400;

export const ONBOARDING_ART_COLOR_TWEEN_MS = COLOR_TWEEN_MS;

export type OnboardingArtHandle = {
  setColors: (background: string, arrowColor: string, durationMs: number, onDone?: () => void) => void;
  destroy: () => void;
};

export type OnboardingArtMountOptions = {
  mode: FirstRunArtScene["mode"];
  panel: false;
  minimalSphere: true;
  allowModeSwitch: false;
  bgColor: string;
  arrowColor: string;
  overrides?: { globe: { count: number } };
};

type OnboardingArtApi = {
  mount: (target: HTMLElement, options: OnboardingArtMountOptions) => OnboardingArtHandle | null;
};

declare global {
  interface Window {
    THREE?: unknown;
    SuperplaneArt?: OnboardingArtApi;
  }
}

export function onboardingArtMountOptions(scene: FirstRunArtScene): OnboardingArtMountOptions {
  const options: OnboardingArtMountOptions = {
    mode: scene.mode,
    panel: false,
    minimalSphere: true,
    allowModeSwitch: false,
    bgColor: scene.background,
    arrowColor: scene.arrowColor,
  };
  if (scene.mode === "globe" && scene.count) {
    options.overrides = { globe: { count: scene.count } };
  }
  return options;
}

export function onboardingArtScriptUrl(file: string, base = import.meta.env.BASE_URL || "/"): string {
  const prefix = base.endsWith("/") ? base : `${base}/`;
  return `${prefix}assets/onboarding/${file}`;
}

function loadScript(src: string): Promise<void> {
  const existing = document.querySelector(`script[data-onboarding-src="${src}"]`);
  if (existing instanceof HTMLScriptElement) {
    if (existing.dataset.loaded === "true") return Promise.resolve();
    if (existing.dataset.failed === "true") return Promise.reject(new Error(`missing ${src}`));
    return new Promise((resolve, reject) => {
      existing.addEventListener("load", () => resolve(), { once: true });
      existing.addEventListener("error", () => reject(new Error(`missing ${src}`)), { once: true });
    });
  }

  return new Promise((resolve, reject) => {
    const script = document.createElement("script");
    script.src = src;
    script.async = false;
    script.dataset.onboardingSrc = src;
    script.onload = () => {
      script.dataset.loaded = "true";
      resolve();
    };
    script.onerror = () => {
      script.dataset.failed = "true";
      reject(new Error(`missing ${src}`));
    };
    try {
      document.head.appendChild(script);
    } catch (error) {
      script.dataset.failed = "true";
      reject(error instanceof Error ? error : new Error(`missing ${src}`));
    }
  });
}

function externalScriptsUnavailable(): boolean {
  return "happyDOM" in window;
}

export async function loadOnboardingArt(): Promise<void> {
  if (window.THREE && window.SuperplaneArt) return;
  if (externalScriptsUnavailable()) return;
  await loadScript(onboardingArtScriptUrl("three.min.js"));
  await loadScript(onboardingArtScriptUrl("superplane-art.js"));
}

export function mountOnboardingArt(target: HTMLElement, scene: FirstRunArtScene): OnboardingArtHandle | null {
  return window.SuperplaneArt?.mount(target, onboardingArtMountOptions(scene)) ?? null;
}

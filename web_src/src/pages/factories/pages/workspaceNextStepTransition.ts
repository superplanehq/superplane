import { flushSync } from "react-dom";

function prefersReducedMotion(): boolean {
  return window.matchMedia?.("(prefers-reduced-motion: reduce)").matches === true;
}

/** Morph the next-steps badge between the banner and the header. */
export function runWorkspaceNextStepTransition(update: () => void): void {
  const startViewTransition = document.startViewTransition?.bind(document);
  if (!startViewTransition || prefersReducedMotion()) {
    update();
    return;
  }
  startViewTransition(() => {
    flushSync(update);
  });
}

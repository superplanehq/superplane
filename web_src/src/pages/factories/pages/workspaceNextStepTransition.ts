import { flushSync } from "react-dom";

function prefersReducedMotion(): boolean {
  return window.matchMedia?.("(prefers-reduced-motion: reduce)").matches === true;
}

/** Animate the next-steps banner and header title. */
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

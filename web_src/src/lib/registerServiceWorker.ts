export function registerServiceWorker(): void {
  if (!("serviceWorker" in navigator)) {
    return;
  }
  if (!import.meta.env.PROD) {
    return;
  }
  window.addEventListener("load", () => {
    navigator.serviceWorker.register("/sw.js", { scope: "/" }).catch(() => {
      // Installability is best-effort. Ignore registration failures.
    });
  });
}

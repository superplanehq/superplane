/* SuperPlane installability service worker.
 * No offline caching. The fetch handler exists so browsers treat the app
 * as installable. All requests pass through to the network.
 */

self.addEventListener("install", () => {
  self.skipWaiting();
});

self.addEventListener("activate", (event) => {
  event.waitUntil(self.clients.claim());
});

self.addEventListener("fetch", (event) => {
  if (event.request.method !== "GET") {
    return;
  }
  event.respondWith(fetch(event.request));
});

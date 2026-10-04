import { describe, expect, it } from "bun:test";

interface SentryWindow extends Window {
  SUPERPLANE_SENTRY_DSN?: string;
}

const COOKIE_VALUE = "nz-cookie-value";
const BODY_SECRET = "nz-body-secret";

function failedRequestWithSecrets(): Promise<Response> {
  return Promise.resolve(
    new Response(JSON.stringify({ password: BODY_SECRET }), {
      status: 500,
      headers: {
        "Content-Type": "application/json",
        "Set-Cookie": `nzpref=${COOKIE_VALUE}`,
      },
    }),
  );
}

describe("sentry report privacy", () => {
  it("omits cookies and request bodies from reports", async () => {
    const sentryWindow = window as SentryWindow;
    const originalFetch = window.fetch;
    sentryWindow.SUPERPLANE_SENTRY_DSN = "https://public@o0.ingest.sentry.io/0";
    Object.defineProperty(failedRequestWithSecrets, "toString", {
      value: () => "function fetch() { [native code] }",
    });
    window.fetch = failedRequestWithSecrets as typeof window.fetch;

    try {
      const { Sentry } = await import("./sentry");
      const reporter = Sentry.getClient();
      expect(reporter).toBeDefined();

      const collected = reporter!.getDataCollectionOptions();
      expect(collected.cookies).toBe(false);
      expect(collected.httpBodies).toEqual([]);

      const captured: Sentry.Event[] = [];
      const probe = Sentry.init({
        dsn: "https://public@o0.ingest.sentry.io/0",
        sendClientReports: false,
        defaultIntegrations: false,
        dataCollection: reporter!.getOptions().dataCollection,
        beforeSend(event) {
          captured.push(event);
          return null;
        },
        integrations: [Sentry.httpClientIntegration()],
      });

      await window.fetch("https://example.test/api/secret", {
        method: "POST",
        headers: {
          Cookie: `nzpref=${COOKIE_VALUE}`,
          "Content-Type": "application/json",
        },
        body: JSON.stringify({ password: BODY_SECRET }),
      });
      await Sentry.flush(2000);

      expect(captured.length).toBeGreaterThan(0);
      const report = JSON.stringify(captured);
      expect(report).not.toContain(COOKIE_VALUE);
      expect(report).not.toContain(BODY_SECRET);
      for (const event of captured) {
        expect(event.request?.cookies).toBeUndefined();
        expect(event.request?.data).toBeUndefined();
        expect(event.contexts?.response?.cookies).toBeUndefined();
      }

      await probe?.close(0);
      await reporter?.close(0);
    } finally {
      window.fetch = originalFetch;
      delete sentryWindow.SUPERPLANE_SENTRY_DSN;
    }
  });
});

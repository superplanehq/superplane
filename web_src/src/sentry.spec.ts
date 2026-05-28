import { describe, expect, it } from "vitest";
import { BROWSER_EXTENSION_DENY_URLS, BROWSER_EXTENSION_IGNORE_PATTERNS } from "@/sentry";

const matchesAny = (patterns: RegExp[], value: string): boolean => patterns.some((pattern) => pattern.test(value));

describe("Sentry browser extension filters", () => {
  describe("ignoreErrors patterns", () => {
    it.each([
      "[1Blocker] Duplicate content script injection blocked",
      "[1blocker] some other warning",
      "[AdGuard] AdGuard message",
      "[uBlock] some uBlock log",
      "AdGuardAssistant initialised",
      "AdblockPlus is enabled",
      "Adblock detected",
      "Ghostery loaded",
      "LastPass autofill",
      "1Password content script ready",
      "Bitwarden content script injected",
      "Grammarly script loaded",
      "MetaMask: connected",
      "Duplicate content script injection blocked",
    ])("matches known browser-extension noise: %s", (message) => {
      expect(matchesAny(BROWSER_EXTENSION_IGNORE_PATTERNS, message)).toBe(true);
    });

    it.each([
      "TypeError: Cannot read properties of undefined (reading 'foo')",
      "Failed to fetch /api/canvases",
      "ReferenceError: bar is not defined",
      "Unhandled promise rejection: NetworkError",
      "Workflow run failed: timeout exceeded",
    ])("does not match application errors: %s", (message) => {
      expect(matchesAny(BROWSER_EXTENSION_IGNORE_PATTERNS, message)).toBe(false);
    });
  });

  describe("denyUrls patterns", () => {
    it.each([
      "chrome://extensions/script.js",
      "chrome-extension://abc123/content.js",
      "moz-extension://abc123/content.js",
      "safari-extension://abc123/content.js",
      "safari-web-extension://abc123/content.js",
      "webkit-masked-url://hidden/",
      "edge://settings/",
    ])("denies extension URL: %s", (url) => {
      expect(matchesAny(BROWSER_EXTENSION_DENY_URLS, url)).toBe(true);
    });

    it.each([
      "https://app.superplane.com/login",
      "https://app.superplane.com/static/main.js",
      "http://localhost:8000/main.tsx",
    ])("does not deny application URL: %s", (url) => {
      expect(matchesAny(BROWSER_EXTENSION_DENY_URLS, url)).toBe(false);
    });
  });
});

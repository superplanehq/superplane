"use strict";

/**
 * HTTP client for the runner API. The run token is the only credential.
 */

function createClient(env = process.env, fetchImpl = fetch) {
  async function request(method, path, body) {
    const baseURL = String(env.SUPERPLANE_BASE_URL || "")
      .trim()
      .replace(/\/$/, "");
    const token = String(env.SUPERPLANE_RUNNER_TOKEN || "").trim();
    if (!baseURL || !token) {
      throw new Error("SUPERPLANE_BASE_URL and SUPERPLANE_RUNNER_TOKEN are required");
    }
    const headers = { Authorization: `Bearer ${token}`, Accept: "application/json" };
    const init = { method, headers };
    if (body !== undefined) {
      headers["Content-Type"] = "application/json";
      init.body = JSON.stringify(body);
    }
    const response = await fetchImpl(`${baseURL}${path}`, init);
    const text = await response.text();
    const parsed = parseBody(text);
    if (!response.ok) {
      const error = new Error(
        String(parsed.message || parsed.error || text || `HTTP ${response.status}`).trim(),
      );
      error.status = response.status;
      throw error;
    }
    return parsed;
  }

  return { request };
}

function parseBody(text) {
  if (!text) return {};
  try {
    return JSON.parse(text);
  } catch {
    return { message: text.trim() };
  }
}

module.exports = { createClient };

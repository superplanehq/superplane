"use strict";

/**
 * HTTP client for the runner API. The run token is the only credential.
 * Tools that read local task files use client.env.
 */

function createClient(env = process.env, fetchImpl = fetch) {
  function connection() {
    const baseURL = String(env.SUPERPLANE_BASE_URL || "")
      .trim()
      .replace(/\/$/, "");
    const token = String(env.SUPERPLANE_RUNNER_TOKEN || "").trim();
    if (!baseURL || !token) {
      throw new Error("SUPERPLANE_BASE_URL and SUPERPLANE_RUNNER_TOKEN are required");
    }
    return { baseURL, token };
  }

  async function send(path, init) {
    const { baseURL, token } = connection();
    const headers = { ...init.headers, Authorization: `Bearer ${token}`, Accept: "application/json" };
    const response = await fetchImpl(`${baseURL}${path}`, { ...init, headers });
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

  async function request(method, path, body) {
    if (body === undefined) return send(path, { method, headers: {} });
    return send(path, {
      method,
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  }

  // upload streams a request body, for example a file read stream.
  async function upload(path, headers, body) {
    return send(path, { method: "POST", headers, body, duplex: "half" });
  }

  return { env, request, upload };
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

"use strict";

/**
 * HTTP client for the runner API. The run token is the only credential.
 * Tools that read local task files use client.env.
 */

// Every request ends before the server's tool call deadline. An upload can
// carry a 100 MB artifact, so it gets more time.
const DEFAULT_TIMEOUTS = { request: 60000, upload: 240000 };

function createClient(env = process.env, fetchImpl = fetch, timeouts = DEFAULT_TIMEOUTS) {
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

  async function send(path, init, timeoutMs) {
    const { baseURL, token } = connection();
    const headers = { ...init.headers, Authorization: `Bearer ${token}`, Accept: "application/json" };
    const controller = new AbortController();
    const timer = setTimeout(() => controller.abort(), timeoutMs);
    let response;
    let text;
    try {
      response = await fetchImpl(`${baseURL}${path}`, { ...init, headers, signal: controller.signal });
      text = await response.text();
    } catch (err) {
      if (controller.signal.aborted) {
        throw new Error(`SuperPlane did not answer ${init.method} ${path} in ${timeoutMs / 1000}s`);
      }
      throw err;
    } finally {
      clearTimeout(timer);
    }
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
    if (body === undefined) return send(path, { method, headers: {} }, timeouts.request);
    return send(
      path,
      {
        method,
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      },
      timeouts.request,
    );
  }

  // upload streams a request body, for example a file read stream.
  async function upload(path, headers, body) {
    return send(path, { method: "POST", headers, body, duplex: "half" }, timeouts.upload);
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

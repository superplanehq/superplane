"use strict";

const { isCompactStatusText } = require("./analysis_protocol");

function readEnv(name) {
  const value = String(process.env[name] || "").trim();
  if (!value) throw new Error(`${name} is required`);
  return value;
}

async function requestJSON(method, requestPath, body) {
  const baseURL = readEnv("SUPERPLANE_BASE_URL").replace(/\/$/, "");
  const token = readEnv("SUPERPLANE_RUN_TOKEN");
  const response = await fetch(`${baseURL}${requestPath}`, {
    method,
    headers: {
      Authorization: `Bearer ${token}`,
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });
  const text = await response.text();
  let parsed = {};
  if (text) {
    try {
      parsed = JSON.parse(text);
    } catch {
      parsed = { message: text };
    }
  }
  if (!response.ok) {
    const error = new Error(parsed.message || parsed.error || text || `HTTP ${response.status}`);
    error.status = response.status;
    throw error;
  }
  return parsed;
}

// The runners call this after a planning turn. It is not an MCP tool.
async function recordAgentMessage(text) {
  const body = String(text || "").trim();
  if (!body || isCompactStatusText(body)) return { status: "ignored" };
  const activityID = String(process.env.SUPERPLANE_ACTIVITY_ID || "").trim();
  return requestJSON("POST", "/api/v1/runner/planning-sessions/agent-messages", {
    text: body,
    ...(activityID ? { activity_id: activityID } : {}),
  });
}

module.exports = { recordAgentMessage };

"use strict";

const MAX_OWNER_PATHS = 10;

function ownerToolsEnabled(env = process.env) {
  return Boolean(String(env.SUPERPLANE_OWNER_TOKEN || "").trim());
}

async function ownerRequest(endpoint, body) {
  const baseURL = String(process.env.SUPERPLANE_BASE_URL || "")
    .trim()
    .replace(/\/$/, "");
  const ownerToken = String(process.env.SUPERPLANE_OWNER_TOKEN || "").trim();
  const token =
    ownerToken || String(process.env.SUPERPLANE_RUN_TOKEN || "").trim();
  if (!baseURL || !token) throw new Error("Task owner capability is required");
  const prefix = ownerToken
    ? "/api/v1/runner/tasks"
    : "/api/v1/runner/planning-sessions";
  const response = await fetch(`${baseURL}${prefix}/${endpoint}`, {
    method: "POST",
    headers: {
      Authorization: `Bearer ${token}`,
      Accept: "application/json",
      "Content-Type": "application/json",
    },
    body: JSON.stringify(body),
  });
  const text = await response.text();
  let result;
  try {
    result = text ? JSON.parse(text) : {};
  } catch {
    result = { message: text };
  }
  if (!response.ok)
    throw new Error(
      result.message || result.error || text || `HTTP ${response.status}`,
    );
  return result;
}

async function suggestOwners(input) {
  const raw = input && Array.isArray(input.paths) ? input.paths : [];
  const paths = raw
    .map((value) => String(value || "").trim())
    .filter(Boolean)
    .slice(0, MAX_OWNER_PATHS);
  return ownerRequest("suggest-owners", { paths });
}

async function assignOwner(input) {
  const user = String((input && input.user) || "").trim();
  const reason = String((input && input.reason) || "").trim();
  if (!user) throw new Error("user is required");
  if (!reason) throw new Error("reason is required");
  return ownerRequest("owner", { user, reason });
}

const SUGGEST_OWNERS_TOOL = {
  name: "suggest_owners",
  description:
    "Find the people who can review or own this task. Returns members of the organization with name, roles, GitHub login, and open_tasks (the open tasks they own in this factory), fewest first, and the current owners of the task. Pass the repository paths the change touches to also get code_owners: members ranked by their recent commits on those paths. Only members with a linked GitHub account can appear in code_owners.",
  inputSchema: {
    type: "object",
    properties: {
      paths: {
        type: "array",
        maxItems: MAX_OWNER_PATHS,
        items: { type: "string" },
        description:
          "Optional. Repository-relative files or directories, most specific first, for example pkg/payments or web_src/src/pages/checkout.tsx.",
      },
    },
  },
};

const ASSIGN_OWNER_TOOL = {
  name: "assign_owner",
  description:
    "Set the selected person as the only owner of this task. Replaces all current owners, including manually assigned owners. Returns assigned with the selected owner. Pass user as a user_id or github_login from suggest_owners.",
  inputSchema: {
    type: "object",
    properties: {
      user: {
        type: "string",
        description: "user_id or github_login from suggest_owners.",
      },
      reason: {
        type: "string",
        description:
          "One sentence that SuperPlane shows on the task timeline, for example: You changed pkg/payments/charge.go 3 times in the last 30 days.",
      },
    },
    required: ["user", "reason"],
  },
};

function ownerTools(env = process.env) {
  return ownerToolsEnabled(env) ? [SUGGEST_OWNERS_TOOL, ASSIGN_OWNER_TOOL] : [];
}

module.exports = {
  ownerTools,
  suggestOwners,
  assignOwner,
  SUGGEST_OWNERS_TOOL,
  ASSIGN_OWNER_TOOL,
};

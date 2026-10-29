"use strict";

const MAX_OWNER_PATHS = 10;

function normalizePaths(raw) {
  const values = Array.isArray(raw) ? raw : [];
  return values
    .map((value) => String(value || "").trim())
    .filter(Boolean)
    .slice(0, MAX_OWNER_PATHS);
}

async function suggestOwners(input, client) {
  return client.request("POST", "/api/v1/runner/task/suggest-owners", {
    paths: normalizePaths(input && input.paths),
  });
}

async function assignOwner(input, client) {
  const user = String((input && input.user) || "").trim();
  const reason = String((input && input.reason) || "").trim();
  if (!user) throw new Error("user is required");
  if (!reason) throw new Error("reason is required");
  return client.request("PUT", "/api/v1/runner/task/owner", { user, reason });
}

const SUGGEST_OWNERS = {
  definition: {
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
  },
  handler: suggestOwners,
};

const ASSIGN_OWNER = {
  definition: {
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
  },
  handler: assignOwner,
};

module.exports = {
  tools: [SUGGEST_OWNERS, ASSIGN_OWNER],
};

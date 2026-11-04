"use strict";

async function listMembers(_input, client) {
  return client.request("GET", "/api/v1/runner/members");
}

const LIST_MEMBERS = {
  definition: {
    name: "list_members",
    description:
      "List the people in the organization, sorted by name. Each member has user_id, name, email, github_login, roles (for example Owner), and open_tasks: the draft and open tasks that the person owns in this workspace. Pass a user_id or github_login to assign_task_owner.",
    inputSchema: { type: "object", properties: {} },
  },
  handler: listMembers,
};

module.exports = {
  tools: [LIST_MEMBERS],
};

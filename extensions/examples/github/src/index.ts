import { defineExtension } from "@superplanehq/sdk";
import { closeIssue } from "./components/close-issue.js";
import { createIssue } from "./components/create-issue.js";
import { github } from "./integrations/github.js";
import { onPush } from "./triggers/on-push.js";

export default defineExtension({
  metadata: {
    id: "examples.github",
    name: "GitHub Example Extension",
    version: "0.1.0",
    description: "Reference extension that creates and closes GitHub issues.",
  },
  integrations: [github],
  components: [createIssue, closeIssue],
  triggers: [onPush],
});

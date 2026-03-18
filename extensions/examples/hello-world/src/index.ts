import { defineExtension } from "@superplanehq/sdk";
import { helloWorld } from "./components/hello-world.js";

export default defineExtension({
  metadata: {
    id: "@examples/hello-world",
    name: "@examples/hello-world",
    version: "0.1.0",
    description: "Example extension that emits a hello world message.",
  },
  components: [helloWorld],
});

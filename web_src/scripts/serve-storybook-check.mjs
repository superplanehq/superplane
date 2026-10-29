import { preview } from "vite";
import path from "node:path";
import { fileURLToPath } from "node:url";

const webRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const server = await preview({
  configFile: false,
  root: webRoot,
  build: { outDir: "storybook-static" },
  preview: {
    host: "127.0.0.1",
    port: Number(process.argv[2]),
    strictPort: true,
  },
});

await new Promise((resolve) => {
  process.once("SIGTERM", resolve);
  process.once("SIGINT", resolve);
});
await server.close();

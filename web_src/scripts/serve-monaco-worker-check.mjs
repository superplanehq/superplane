import { createServer } from "node:http";
import { readFile, readdir, stat, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { build } from "vite";

const webRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const outDir = path.resolve(webRoot, "../tmp/monaco-worker-check");
const entryId = "virtual:monaco-worker-startup-entry";
const resolvedEntryId = `\0${entryId}`;
const readyPath = process.argv[2];
if (!readyPath) {
  throw new Error("usage: node scripts/serve-monaco-worker-check.mjs <ready-file>");
}

const pageHtml = `<!doctype html>
<meta charset="utf-8">
<title>Monaco worker startup</title>
`;

await build({
  configFile: path.join(webRoot, "vite.config.ts"),
  root: webRoot,
  mode: "production",
  plugins: [
    {
      name: "monaco-worker-startup-entry",
      resolveId(id) {
        if (id === entryId) {
          return resolvedEntryId;
        }
        return undefined;
      },
      load(id) {
        if (id !== resolvedEntryId) {
          return undefined;
        }
        return [
          'import editorWorkerUrl from "monaco-editor/esm/vs/editor/editor.worker?worker&url";',
          "export default editorWorkerUrl;",
        ].join("\n");
      },
    },
  ],
  build: {
    outDir,
    emptyOutDir: true,
    sourcemap: false,
    rolldownOptions: {
      input: entryId,
    },
  },
});

const workerFile = await findEditorWorker(outDir);
const workerServer = await listen((request, response) => {
  serveFile(outDir, request.url ?? "/", response, true);
});
const pageServer = await listen((_request, response) => {
  response.writeHead(200, { "Content-Type": "text/html; charset=utf-8" });
  response.end(pageHtml);
});

const workerPath = `/${path.relative(outDir, workerFile).split(path.sep).join("/")}`;
const ready = {
  pageUrl: `http://127.0.0.1:${pageServer.port}/`,
  workerUrl: `http://127.0.0.1:${workerServer.port}${workerPath}`,
};
await writeFile(readyPath, `${JSON.stringify(ready)}\n`);

await new Promise((resolve) => {
  process.once("SIGTERM", resolve);
  process.once("SIGINT", resolve);
});
workerServer.server.close();
pageServer.server.close();

async function findEditorWorker(directory) {
  const matches = [];
  await walk(directory, matches);
  if (matches.length !== 1) {
    throw new Error(`expected one editor.worker script in ${directory}, found ${matches.length}: ${matches.join(", ")}`);
  }
  return matches[0];
}

async function walk(directory, matches) {
  const entries = await readdir(directory, { withFileTypes: true });
  for (const entry of entries) {
    const entryPath = path.join(directory, entry.name);
    if (entry.isDirectory()) {
      await walk(entryPath, matches);
      continue;
    }
    if (entry.isFile() && /^editor\.worker-.*\.js$/.test(entry.name)) {
      matches.push(entryPath);
    }
  }
}

function listen(handler) {
  return new Promise((resolve, reject) => {
    const server = createServer(handler);
    server.once("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const address = server.address();
      if (!address || typeof address === "string") {
        reject(new Error("server did not bind a TCP port"));
        return;
      }
      resolve({ server, port: address.port });
    });
  });
}

async function serveFile(root, requestUrl, response, allowCrossOrigin) {
  if (allowCrossOrigin) {
    response.setHeader("Access-Control-Allow-Origin", "*");
  }

  const pathname = decodeURIComponent(new URL(requestUrl, "http://127.0.0.1").pathname);
  const relativePath = pathname.replace(/^\/+/, "");
  const filePath = path.resolve(root, relativePath);
  if (filePath !== root && !filePath.startsWith(`${root}${path.sep}`)) {
    response.writeHead(403);
    response.end("forbidden");
    return;
  }

  try {
    const fileStat = await stat(filePath);
    if (!fileStat.isFile()) {
      response.writeHead(404);
      response.end("not found");
      return;
    }
    const body = await readFile(filePath);
    response.writeHead(200, {
      "Content-Type": "text/javascript; charset=utf-8",
      "Content-Length": body.length,
    });
    response.end(body);
  } catch {
    response.writeHead(404);
    response.end("not found");
  }
}

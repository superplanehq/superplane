import * as path from "path";
import * as vm from "vm";
import { RpcTransport } from "./rpc";
import * as sdk from "./sdk";
import {
  PluginContextImpl,
  buildExecutionContext,
  buildSetupContext,
  buildTriggerSetupContext,
  buildWebhookContext,
  buildIntegrationSyncContext,
  buildIntegrationRequestContext,
  buildIntegrationCleanupContext,
  buildWebhookHandlerSetupContext,
  buildWebhookHandlerCleanupContext,
} from "./sdk";

/**
 * Converts ES module syntax to CommonJS so scripts can run inside vm.compileFunction.
 * Handles: import, export function, export async function, export const/let/var, export default.
 */
function esmToCjs(source: string): string {
  const exportedNames: string[] = [];

  let result = source
    // import { a, b } from "mod"  →  const { a, b } = require("mod")
    .replace(/\bimport\s+\{([^}]+)\}\s+from\s+["']([^"']+)["']\s*;?/g,
      (_match, names, mod) => `const {${names}} = require("${mod}");`)
    // import X from "mod"  →  const X = require("mod")
    .replace(/\bimport\s+(\w+)\s+from\s+["']([^"']+)["']\s*;?/g,
      (_match, name, mod) => `const ${name} = require("${mod}");`)
    // import * as X from "mod"  →  const X = require("mod")
    .replace(/\bimport\s+\*\s+as\s+(\w+)\s+from\s+["']([^"']+)["']\s*;?/g,
      (_match, name, mod) => `const ${name} = require("${mod}");`)
    // export async function name(...)
    .replace(/\bexport\s+async\s+function\s+(\w+)/g, (_match, name) => {
      exportedNames.push(name);
      return `async function ${name}`;
    })
    // export function name(...)
    .replace(/\bexport\s+function\s+(\w+)/g, (_match, name) => {
      exportedNames.push(name);
      return `function ${name}`;
    })
    // export const/let/var name
    .replace(/\bexport\s+(const|let|var)\s+(\w+)/g, (_match, kind, name) => {
      exportedNames.push(name);
      return `${kind} ${name}`;
    })
    // export default (treat as module.exports directly)
    .replace(/\bexport\s+default\s+/g, "module.exports = ");

  if (exportedNames.length > 0) {
    result += "\n" + exportedNames.map((n) => `module.exports.${n} = ${n};`).join("\n") + "\n";
  }

  return result;
}

interface LoadedPlugin {
  id: string;
  path: string;
  context: PluginContextImpl;
  deactivate?: () => void | Promise<void>;
}

const plugins = new Map<string, LoadedPlugin>();

// Pre-populate the require cache so that `require("@superplane/sdk")`
// resolves to the SDK bundled in the Plugin Host.
// This avoids monkey-patching Module._resolveFilename which is
// read-only in Node.js v22+.
const sdkCacheKey = require.resolve("./sdk");
require.cache["@superplane/sdk"] = require.cache[sdkCacheKey]!;

const rpc = new RpcTransport(async (method: string, params: any) => {
  switch (method) {
    case "plugin/activate":
      return handleActivate(params);

    case "plugin/deactivate":
      return handleDeactivate(params);

    case "plugin/activateInline":
      return handleActivateInline(params);

    default: {
      if (method.startsWith("component/")) {
        return handleComponentCall(method, params);
      }
      if (method.startsWith("trigger/")) {
        return handleTriggerCall(method, params);
      }
      if (method.startsWith("integration/")) {
        return handleIntegrationCall(method, params);
      }
      if (method.startsWith("webhookHandler/")) {
        return handleWebhookHandlerCall(method, params);
      }

      throw new Error(`Unknown method: ${method}`);
    }
  }
});

async function handleActivate(params: {
  pluginId: string;
  pluginPath: string;
}): Promise<void> {
  const { pluginId, pluginPath } = params;

  if (plugins.has(pluginId)) {
    return;
  }

  const extensionPath = path.resolve(pluginPath, "extension.js");
  const context = new PluginContextImpl(pluginId, rpc);

  let pluginModule: any;
  try {
    pluginModule = require(extensionPath);
  } catch (err: any) {
    throw new Error(
      `Failed to load plugin ${pluginId}: ${err.message}`
    );
  }

  if (typeof pluginModule.activate !== "function") {
    throw new Error(
      `Plugin ${pluginId} does not export an activate() function`
    );
  }

  await pluginModule.activate(context);

  const plugin: LoadedPlugin = {
    id: pluginId,
    path: pluginPath,
    context,
    deactivate:
      typeof pluginModule.deactivate === "function"
        ? pluginModule.deactivate
        : undefined,
  };

  plugins.set(pluginId, plugin);
}

async function handleActivateInline(params: {
  pluginId: string;
  source: string;
  manifest: any;
}): Promise<{ components: any[]; triggers: any[] }> {
  const { pluginId, source } = params;

  if (plugins.has(pluginId)) {
    await handleDeactivate({ pluginId });
  }

  const context = new PluginContextImpl(pluginId, rpc);

  let pluginModule: any;
  try {
    const cjsSource = esmToCjs(source);
    const moduleExports: any = {};
    const moduleObj = { exports: moduleExports };

    const scriptRequire = (id: string) => {
      if (id === "@superplane/sdk") return sdk;
      return require(id);
    };

    const wrappedFn = vm.compileFunction(cjsSource, ["require", "module", "exports"], {
      filename: `${pluginId}.js`,
    });
    wrappedFn(scriptRequire, moduleObj, moduleExports);
    pluginModule = moduleObj.exports;
  } catch (err: any) {
    throw new Error(`Failed to evaluate script ${pluginId}: ${err.message}`);
  }

  if (typeof pluginModule.activate !== "function") {
    throw new Error(`Script ${pluginId} does not export an activate() function`);
  }

  await pluginModule.activate(context);

  plugins.set(pluginId, {
    id: pluginId,
    path: "",
    context,
    deactivate:
      typeof pluginModule.deactivate === "function" ? pluginModule.deactivate : undefined,
  });

  return {
    components: context.components.getRegisteredMetadata(),
    triggers: context.triggers.getRegisteredMetadata(),
  };
}

async function handleDeactivate(params: {
  pluginId: string;
}): Promise<void> {
  const plugin = plugins.get(params.pluginId);
  if (!plugin) return;

  if (plugin.deactivate) {
    await plugin.deactivate();
  }

  // Dispose all tracked subscriptions
  for (const sub of plugin.context.subscriptions) {
    sub.dispose();
  }

  plugins.delete(params.pluginId);
}

async function handleComponentCall(
  method: string,
  params: { pluginId: string; component: string; context: any }
): Promise<any> {
  const plugin = plugins.get(params.pluginId);
  if (!plugin) {
    throw new Error(`Plugin ${params.pluginId} is not activated`);
  }

  const handler = plugin.context.components.getHandler(params.component);
  if (!handler) {
    throw new Error(
      `Component handler ${params.component} not registered by plugin ${params.pluginId}`
    );
  }

  const action = method.replace("component/", "");

  switch (action) {
    case "setup": {
      const ctx = buildSetupContext(params.context, rpc, params.pluginId);
      await handler.setup?.(ctx);
      return null;
    }

    case "execute": {
      const { ctx, getResult } = buildExecutionContext(
        params.context,
        rpc,
        params.pluginId
      );
      await handler.execute(ctx);
      return getResult();
    }

    case "cancel": {
      const { ctx } = buildExecutionContext(
        params.context,
        rpc,
        params.pluginId
      );
      await handler.cancel?.(ctx);
      return null;
    }

    case "cleanup": {
      const ctx = buildSetupContext(params.context, rpc, params.pluginId);
      await handler.cleanup?.(ctx);
      return null;
    }

    default:
      throw new Error(`Unknown component action: ${action}`);
  }
}

async function handleTriggerCall(
  method: string,
  params: { pluginId: string; trigger: string; context: any }
): Promise<any> {
  const plugin = plugins.get(params.pluginId);
  if (!plugin) {
    throw new Error(`Plugin ${params.pluginId} is not activated`);
  }

  const handler = plugin.context.triggers.getHandler(params.trigger);
  if (!handler) {
    throw new Error(
      `Trigger handler ${params.trigger} not registered by plugin ${params.pluginId}`
    );
  }

  const action = method.replace("trigger/", "");

  switch (action) {
    case "setup": {
      const ctx = buildTriggerSetupContext(
        params.context,
        rpc,
        params.pluginId
      );
      await handler.setup?.(ctx);
      return null;
    }

    case "handleWebhook": {
      const ctx = buildWebhookContext(params.context, rpc, params.pluginId);
      const result = await handler.handleWebhook?.(ctx);
      return result ?? { status: 200 };
    }

    case "cleanup": {
      const ctx = buildTriggerSetupContext(
        params.context,
        rpc,
        params.pluginId
      );
      await handler.cleanup?.(ctx);
      return null;
    }

    default:
      throw new Error(`Unknown trigger action: ${action}`);
  }
}

async function handleIntegrationCall(
  method: string,
  params: { pluginId: string; integration: string; context: any }
): Promise<any> {
  const plugin = plugins.get(params.pluginId);
  if (!plugin) {
    throw new Error(`Plugin ${params.pluginId} is not activated`);
  }

  const handler = plugin.context.integrations.getHandler(params.integration);
  if (!handler) {
    throw new Error(
      `Integration handler ${params.integration} not registered by plugin ${params.pluginId}`
    );
  }

  const action = method.replace("integration/", "");

  switch (action) {
    case "sync": {
      const ctx = buildIntegrationSyncContext(
        params.context,
        rpc,
        params.pluginId
      );
      await handler.sync?.(ctx);
      return null;
    }

    case "handleRequest": {
      const ctx = buildIntegrationRequestContext(
        params.context,
        rpc,
        params.pluginId
      );
      return (await handler.handleRequest?.(ctx)) ?? null;
    }

    case "cleanup": {
      const ctx = buildIntegrationCleanupContext(
        params.context,
        rpc,
        params.pluginId
      );
      await handler.cleanup?.(ctx);
      return null;
    }

    default:
      throw new Error(`Unknown integration action: ${action}`);
  }
}

async function handleWebhookHandlerCall(
  method: string,
  params: { pluginId: string; integration: string; context: any }
): Promise<any> {
  const plugin = plugins.get(params.pluginId);
  if (!plugin) {
    throw new Error(`Plugin ${params.pluginId} is not activated`);
  }

  const handler = plugin.context.integrations.getHandler(params.integration);
  if (!handler?.webhookHandler) {
    throw new Error(
      `Webhook handler not registered for integration ${params.integration} by plugin ${params.pluginId}`
    );
  }

  const action = method.replace("webhookHandler/", "");

  switch (action) {
    case "setup": {
      const ctx = buildWebhookHandlerSetupContext(
        params.context,
        rpc,
        params.pluginId
      );
      return await handler.webhookHandler.setup(ctx);
    }

    case "cleanup": {
      const ctx = buildWebhookHandlerCleanupContext(
        params.context,
        rpc,
        params.pluginId
      );
      await handler.webhookHandler.cleanup(ctx);
      return null;
    }

    case "compareConfig": {
      return await handler.webhookHandler.compareConfig(
        params.context.a,
        params.context.b
      );
    }

    default:
      throw new Error(`Unknown webhookHandler action: ${action}`);
  }
}

// Catch unhandled errors to prevent crashing the Plugin Host
process.on("uncaughtException", (err) => {
  process.stderr.write(`Plugin Host uncaught exception: ${err.message}\n`);
  process.stderr.write(`${err.stack}\n`);
});

process.on("unhandledRejection", (reason) => {
  process.stderr.write(`Plugin Host unhandled rejection: ${reason}\n`);
});

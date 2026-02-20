import * as path from "path";
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

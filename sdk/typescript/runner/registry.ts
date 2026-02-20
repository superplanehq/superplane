import path from "node:path";
import { pathToFileURL } from "node:url";
import type {
  ComponentImplementation,
  IntegrationImplementation,
  TriggerImplementation,
} from "../types.ts";

type IntegrationManifest = {
  name?: string;
  components?: Array<{ name?: string; directory?: string }>;
  triggers?: Array<{ name?: string; directory?: string }>;
};

type ComponentManifest = {
  name?: string;
};

export type CapabilityKind = "component" | "integration" | "trigger";

export type Capability = {
  kind: CapabilityKind;
  name: string;
  operations: string[];
  schemaHash: string;
};

export class ModuleRegistry {
  readonly components = new Map<string, ComponentImplementation>();
  readonly integrations = new Map<string, IntegrationImplementation>();
  readonly triggers = new Map<string, TriggerImplementation>();

  static async fromEnv(): Promise<ModuleRegistry> {
    const registry = new ModuleRegistry();

    await registry.loadComponentsFromDir(Deno.env.get("TYPESCRIPT_COMPONENTS_DIR") ?? "");
    await registry.loadIntegrationsFromDir(Deno.env.get("TYPESCRIPT_INTEGRATIONS_DIR") ?? "");

    return registry;
  }

  listCapabilities(): Capability[] {
    const capabilities: Capability[] = [];

    for (const name of this.components.keys()) {
      capabilities.push({
        kind: "component",
        name,
        operations: ["setup", "execute"],
        schemaHash: "v1",
      });
    }

    for (const name of this.integrations.keys()) {
      capabilities.push({
        kind: "integration",
        name,
        operations: ["sync", "cleanup"],
        schemaHash: "v1",
      });
    }

    for (const name of this.triggers.keys()) {
      capabilities.push({
        kind: "trigger",
        name,
        operations: ["setup"],
        schemaHash: "v1",
      });
    }

    capabilities.sort((a, b) => `${a.kind}:${a.name}`.localeCompare(`${b.kind}:${b.name}`));

    return capabilities;
  }

  private async loadComponentsFromDir(baseDir: string): Promise<void> {
    const root = baseDir.trim();
    if (!root) {
      return;
    }

    for await (const entry of Deno.readDir(root)) {
      if (!entry.isDirectory) {
        continue;
      }

      const componentDir = path.join(root, entry.name);
      const manifestPath = path.join(componentDir, "manifest.json");
      const modulePath = path.join(componentDir, "index.ts");

      const manifest = await readJSON<ComponentManifest>(manifestPath);
      const componentName = (manifest.name ?? entry.name).trim();
      if (!componentName) {
        continue;
      }

      const implementation = await importComponent(modulePath);
      this.components.set(componentName, implementation);
    }
  }

  private async loadIntegrationsFromDir(baseDir: string): Promise<void> {
    const root = baseDir.trim();
    if (!root) {
      return;
    }

    for await (const entry of Deno.readDir(root)) {
      if (!entry.isDirectory) {
        continue;
      }

      const integrationDir = path.join(root, entry.name);
      const manifestPath = path.join(integrationDir, "manifest.json");
      const modulePath = path.join(integrationDir, "index.ts");

      const manifest = await readJSON<IntegrationManifest>(manifestPath);
      const integrationName = (manifest.name ?? entry.name).trim();
      if (!integrationName) {
        continue;
      }

      this.integrations.set(integrationName, await importIntegration(modulePath));

      for (const componentRef of manifest.components ?? []) {
        const componentDir = path.join(integrationDir, componentRef.directory ?? "");
        const componentManifest = await readJSON<ComponentManifest>(path.join(componentDir, "manifest.json"));
        const leafName = (componentRef.name ?? componentManifest.name ?? "").trim();
        if (!leafName) {
          continue;
        }

        const fullName = leafName.includes(".") ? leafName : `${integrationName}.${leafName}`;
        this.components.set(fullName, await importComponent(path.join(componentDir, "index.ts")));
      }

      for (const triggerRef of manifest.triggers ?? []) {
        const triggerDir = path.join(integrationDir, triggerRef.directory ?? "");
        const leafName = (triggerRef.name ?? "").trim();
        if (!leafName) {
          continue;
        }

        const fullName = leafName.includes(".") ? leafName : `${integrationName}.${leafName}`;
        this.triggers.set(fullName, await importTrigger(path.join(triggerDir, "index.ts")));
      }
    }
  }
}

async function readJSON<T>(filePath: string): Promise<T> {
  const raw = await Deno.readTextFile(filePath);
  return JSON.parse(raw) as T;
}

async function importComponent(modulePath: string): Promise<ComponentImplementation> {
  const moduleURL = pathToFileURL(path.resolve(modulePath)).href;
  const loaded = (await import(moduleURL)) as { component?: ComponentImplementation };
  if (!loaded.component) {
    throw new Error(`Missing exported component in ${modulePath}`);
  }

  return loaded.component;
}

async function importIntegration(modulePath: string): Promise<IntegrationImplementation> {
  const moduleURL = pathToFileURL(path.resolve(modulePath)).href;
  const loaded = (await import(moduleURL)) as { integration?: IntegrationImplementation };
  if (!loaded.integration) {
    throw new Error(`Missing exported integration in ${modulePath}`);
  }

  return loaded.integration;
}

async function importTrigger(modulePath: string): Promise<TriggerImplementation> {
  const moduleURL = pathToFileURL(path.resolve(modulePath)).href;
  const loaded = (await import(moduleURL)) as { trigger?: TriggerImplementation };
  if (!loaded.trigger) {
    throw new Error(`Missing exported trigger in ${modulePath}`);
  }

  return loaded.trigger;
}

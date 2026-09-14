import type { EventStateRegistry, TriggerRenderer } from "./types";

type MapperLookups = {
  getState: (componentName: string) => EventStateRegistry["getState"];
  getStateMap: (componentName: string) => EventStateRegistry["stateMap"];
  getTriggerRenderer: (name: string) => TriggerRenderer;
};

type ImportMetaWithRequire = ImportMeta & {
  require?: (id: string) => MapperLookups;
};

let lookups: MapperLookups | undefined;

export function registerMapperLookups(next: MapperLookups): void {
  lookups = next;
}

function loadMapperLookups(): MapperLookups {
  // Tests often import a mapper file without the registry barrel. Load the
  // barrel on first use via Bun's ESM require so mapper modules can finish
  // initializing before the registry evaluates every integration.
  const bunRequire = (import.meta as ImportMetaWithRequire).require;
  if (!bunRequire) {
    throw new Error("Mapper registries are not initialized");
  }

  return bunRequire("./index");
}

function mapperLookups(): MapperLookups {
  lookups ??= loadMapperLookups();
  return lookups;
}

export function getState(componentName: string) {
  return mapperLookups().getState(componentName);
}

export function getStateMap(componentName: string) {
  return mapperLookups().getStateMap(componentName);
}

export function getTriggerRenderer(name: string) {
  return mapperLookups().getTriggerRenderer(name);
}

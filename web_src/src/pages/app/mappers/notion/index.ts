import { buildActionStateRegistry } from "../eventDisplay";
import type { ComponentBaseMapper, EventStateRegistry } from "../types";
import { createPageMapper } from "./create_page";
import { searchMapper } from "./search";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  search: searchMapper,
  createPage: createPageMapper,
};

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  search: buildActionStateRegistry("searched"),
  createPage: buildActionStateRegistry("created"),
};

import { ComponentBaseMapper, TriggerRenderer, EventStateRegistry } from "../types";
import { queryGraphQLMapper } from "./query_graphql";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  queryGraphQL: queryGraphQLMapper,
};

export const triggerRenderers: Record<string, TriggerRenderer> = {};

export const eventStateRegistry: Record<string, EventStateRegistry> = {};

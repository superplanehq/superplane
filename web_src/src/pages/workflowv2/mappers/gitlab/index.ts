import { ComponentBaseMapper, EventStateRegistry } from "../types";
import { buildActionStateRegistry } from "../utils";
import { createIssueMapper } from "./component";

export const eventStateRegistry: Record<string, EventStateRegistry> = {
  createIssue: buildActionStateRegistry("created"),
};

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: createIssueMapper,
};

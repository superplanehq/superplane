import { ComponentBaseMapper } from "../types";
import { createIssueMapper } from "./component";

export const componentMappers: Record<string, ComponentBaseMapper> = {
  createIssue: createIssueMapper,
};

import { PlaneletServer } from "./server.js";
import type {
  ActionDefinition,
  PlaneletOptions,
  FieldDefinition,
  FieldOption,
} from "./types.js";

export function createPlanelet(options: PlaneletOptions): PlaneletBuilder {
  return new PlaneletBuilder(options);
}

class PlaneletBuilder {
  private server: PlaneletServer;

  constructor(options: PlaneletOptions) {
    this.server = new PlaneletServer(options);
  }

  action<TParams = Record<string, unknown>>(
    name: string,
    definition: ActionDefinition<TParams>,
  ): this {
    this.server.addAction(name, definition as ActionDefinition);
    return this;
  }

  listen(port: number, callback?: () => void): void {
    this.server.listen(port, callback);
  }
}

export type { ActionDefinition, PlaneletOptions, FieldDefinition, FieldOption };

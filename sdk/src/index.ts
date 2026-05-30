import { PluginServer } from "./server.js";
import type {
  ActionDefinition,
  PluginOptions,
  FieldDefinition,
  FieldOption,
} from "./types.js";

export function createPlugin(options: PluginOptions): PluginBuilder {
  return new PluginBuilder(options);
}

class PluginBuilder {
  private server: PluginServer;

  constructor(options: PluginOptions) {
    this.server = new PluginServer(options);
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

export type { ActionDefinition, PluginOptions, FieldDefinition, FieldOption };

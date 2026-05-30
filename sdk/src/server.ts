import express, { type Express, type Request, type Response } from "express";
import type {
  ActionDefinition,
  ActionManifest,
  ExecuteRequest,
  ExecuteResponse,
  Manifest,
  PluginOptions,
} from "./types.js";

export class PluginServer {
  private app: Express;
  private actions: Map<string, ActionDefinition> = new Map();
  private options: PluginOptions;

  constructor(options: PluginOptions) {
    this.options = options;
    this.app = express();
    this.app.use(express.json());
    this.setupRoutes();
  }

  addAction(name: string, definition: ActionDefinition): void {
    this.actions.set(name, definition);
  }

  listen(port: number, callback?: () => void): void {
    this.app.listen(
      port,
      callback ??
        (() => {
          console.log(
            `Plugin server "${this.options.name}" listening on port ${port}`,
          );
        }),
    );
  }

  getManifest(): Manifest {
    const actions: ActionManifest[] = [];

    for (const [name, def] of this.actions) {
      const fields = Object.entries(def.fields).map(([fieldName, field]) => ({
        name: fieldName,
        label: field.label,
        type: field.type,
        description: field.description ?? "",
        required: field.required ?? false,
        default: field.default,
        options: field.options,
      }));

      actions.push({
        name,
        label: def.label,
        description: def.description ?? "",
        fields,
      });
    }

    return {
      name: this.options.name,
      label: this.options.label ?? this.options.name,
      icon: this.options.icon ?? "puzzle",
      description: this.options.description ?? "",
      actions,
    };
  }

  private setupRoutes(): void {
    this.app.get("/manifest", (_req: Request, res: Response) => {
      res.json(this.getManifest());
    });

    this.app.post(
      "/actions/:name/execute",
      async (req: Request, res: Response) => {
        const name = req.params.name as string;
        const action = this.actions.get(name);

        if (!action) {
          const response: ExecuteResponse = {
            success: false,
            error: `Action "${name}" not found`,
          };
          res.status(404).json(response);
          return;
        }

        const body = req.body as ExecuteRequest;

        try {
          const result = await action.execute(body.parameters ?? {}, {
            input: body.input,
          });

          const response: ExecuteResponse = {
            success: true,
            data: result,
          };
          res.json(response);
        } catch (err) {
          const response: ExecuteResponse = {
            success: false,
            error: err instanceof Error ? err.message : String(err),
          };
          res.status(500).json(response);
        }
      },
    );
  }
}

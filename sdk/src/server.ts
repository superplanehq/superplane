import express, { type Express, type Request, type Response } from "express";
import type {
  ActionDefinition,
  ActionManifest,
  CleanupTriggerRequest,
  CleanupTriggerResponse,
  ExecuteRequest,
  ExecuteResponse,
  HandleTriggerWebhookRequest,
  HandleTriggerWebhookResponse,
  Manifest,
  ParameterDefinition,
  ParameterManifest,
  PlaneletOptions,
  SetupTriggerRequest,
  SetupTriggerResponse,
  TriggerDefinition,
  TriggerManifest,
} from "./types.js";

export class PlaneletServer {
  private app: Express;
  private actions: Map<string, ActionDefinition> = new Map();
  private triggers: Map<string, TriggerDefinition> = new Map();
  private options: PlaneletOptions;

  constructor(options: PlaneletOptions) {
    this.options = options;
    this.app = express();
    this.app.use(express.json());
    this.setupRoutes();
  }

  addAction(id: string, definition: ActionDefinition): void {
    this.actions.set(id, definition);
  }

  addTrigger(id: string, definition: TriggerDefinition): void {
    this.triggers.set(id, definition);
  }

  listen(port: number, callback?: () => void): void {
    this.app.listen(
      port,
      callback ??
        (() => {
          console.log(
            `Planelet server "${this.options.id}" listening on port ${port}`,
          );
        }),
    );
  }

  getManifest(): Manifest {
    const actions: ActionManifest[] = [];
    const triggers: TriggerManifest[] = [];

    for (const [id, def] of this.actions) {
      actions.push({
        id,
        label: def.label,
        icon: def.icon,
        iconUrl: def.iconUrl,
        description: def.description,
        parameters: serializeParameters(def.parameters),
      });
    }

    for (const [id, def] of this.triggers) {
      triggers.push({
        id,
        label: def.label,
        icon: def.icon,
        iconUrl: def.iconUrl,
        description: def.description,
        parameters: serializeParameters(def.parameters),
      });
    }

    return {
      id: this.options.id,
      label: this.options.label ?? this.options.id,
      icon: this.options.icon,
      iconUrl: this.options.iconUrl,
      description: this.options.description,
      actions,
      triggers,
    };
  }

  private setupRoutes(): void {
    this.app.get("/manifest", (_req: Request, res: Response) => {
      res.json(this.getManifest());
    });

    this.app.post(
      "/actions/:id/execute",
      async (req: Request, res: Response) => {
        const id = req.params.id as string;
        const action = this.actions.get(id);

        if (!action) {
          const response: ExecuteResponse = {
            success: false,
            error: `Action "${id}" not found`,
          };
          res.status(404).json(response);
          return;
        }

        const body = req.body as ExecuteRequest;

        try {
          const result = await action.execute({
            parameters: body.parameters ?? {},
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

    this.app.post(
      "/triggers/:id/setup",
      async (req: Request, res: Response) => {
        const id = req.params.id as string;
        const trigger = this.triggers.get(id);

        if (!trigger) {
          const response: SetupTriggerResponse = {
            success: false,
            error: `Trigger "${id}" not found`,
          };
          res.status(404).json(response);
          return;
        }

        const body = req.body as SetupTriggerRequest;

        try {
          const metadata = await trigger.setup({
            parameters: body.parameters ?? {},
            webhook: body.webhook,
          });

          const response: SetupTriggerResponse = {
            success: true,
            metadata: metadata as Record<string, unknown> | undefined,
          };
          res.json(response);
        } catch (err) {
          const response: SetupTriggerResponse = {
            success: false,
            error: err instanceof Error ? err.message : String(err),
          };
          res.status(500).json(response);
        }
      },
    );

    this.app.post(
      "/triggers/:id/cleanup",
      async (req: Request, res: Response) => {
        const id = req.params.id as string;
        const trigger = this.triggers.get(id);

        if (!trigger) {
          const response: CleanupTriggerResponse = {
            success: false,
            error: `Trigger "${id}" not found`,
          };
          res.status(404).json(response);
          return;
        }

        const body = req.body as CleanupTriggerRequest;

        try {
          await trigger.cleanup?.({
            parameters: body.parameters ?? {},
            metadata: body.metadata,
          });

          const response: CleanupTriggerResponse = { success: true };
          res.json(response);
        } catch (err) {
          const response: CleanupTriggerResponse = {
            success: false,
            error: err instanceof Error ? err.message : String(err),
          };
          res.status(500).json(response);
        }
      },
    );

    this.app.post(
      "/triggers/:id/webhook",
      async (req: Request, res: Response) => {
        const id = req.params.id as string;
        const trigger = this.triggers.get(id);

        if (!trigger) {
          const response: HandleTriggerWebhookResponse = {
            success: false,
            error: `Trigger "${id}" not found`,
          };
          res.status(404).json(response);
          return;
        }

        const body = req.body as HandleTriggerWebhookRequest;

        try {
          const result = await trigger.handleWebhook({
            parameters: body.parameters ?? {},
            metadata: body.metadata,
            request: body.request,
          });

          const response: HandleTriggerWebhookResponse = {
            success: true,
            emit: result.emit ?? true,
            eventType: "eventType" in result ? result.eventType : undefined,
            payload: "payload" in result ? result.payload : undefined,
            reason: "reason" in result ? result.reason : undefined,
            response: result.response,
          };
          res.json(response);
        } catch (err) {
          const response: HandleTriggerWebhookResponse = {
            success: false,
            error: err instanceof Error ? err.message : String(err),
          };
          res.status(500).json(response);
        }
      },
    );
  }
}

function serializeParameters(
  parameters: Record<string, ParameterDefinition> = {},
): ParameterManifest[] {
  return Object.entries(parameters).map(([id, parameter]) => ({
    id,
    label: parameter.label,
    type: parameter.type,
    description: parameter.description,
    required: parameter.required ?? false,
    default: parameter.default,
    options: parameter.options,
  }));
}

import { canvasesCommitCanvasStaging, canvasesPutCanvasStaging } from "@/api-client";
import { withOrganizationHeader } from "@/lib/withOrganizationHeader";
import { encodeRepositoryFileContent } from "@/pages/app/files/lib/repository-files";
import { CANVAS_YAML_PATH } from "@/pages/app/lib/workflow-spec-paths";
import { isCanvasNameAlreadyExistsError, uniqueCanvasName } from "@/pages/home/uniqueCanvasName";

import {
  buildIntakeAutomationYaml,
  intakeAutomationDescription,
  intakeAutomationName,
} from "./intakeAutomationTemplate";
import type { LineIntakeSourceId } from "./lineIntakeModel";

const MAX_NAME_RETRY_ATTEMPTS = 20;

interface CreateCanvasResult {
  data?: {
    canvas?: {
      metadata?: {
        id?: string;
      };
    };
  };
}

interface CreateIntakeAutomationInput {
  factoryId: string;
  sourceId: LineIntakeSourceId;
  confidencePct: number;
  createCanvas: (input: {
    name: string;
    description: string;
    factoryId: string;
    method: "template";
  }) => Promise<CreateCanvasResult>;
  /** Names already taken in the org, used to pick "GitHub issues (2)", … */
  existingCanvasNames?: Iterable<string>;
  stageCanvas?: (canvasId: string, canvasYaml: string) => Promise<unknown>;
  commitCanvas?: (canvasId: string) => Promise<unknown>;
  deleteCanvas?: (canvasId: string) => Promise<unknown>;
}

export async function createIntakeAutomation({
  factoryId,
  sourceId,
  confidencePct,
  createCanvas,
  existingCanvasNames,
  stageCanvas = stageIntakeCanvas,
  commitCanvas = commitIntakeCanvas,
  deleteCanvas,
}: CreateIntakeAutomationInput): Promise<string> {
  const canvasId = await createIntakeCanvasShell({
    factoryId,
    sourceId,
    createCanvas,
    existingCanvasNames,
  });

  try {
    await stageCanvas(canvasId, buildIntakeAutomationYaml(sourceId, confidencePct));
    // The intake graph must be live: staged nodes are invisible to the factory
    // apps API, so an uncommitted canvas never appears as an intake source.
    await commitCanvas(canvasId);
  } catch (error) {
    try {
      await deleteCanvas?.(canvasId);
    } catch {
      // Keep the original error because it explains why intake creation failed.
    }
    throw error;
  }
  return canvasId;
}

// An intake canvas can already exist under the same name, for example when an
// earlier attempt left an automation behind. Canvas names are unique per
// organization, so fall back to a numbered name instead of failing.
async function createIntakeCanvasShell({
  factoryId,
  sourceId,
  createCanvas,
  existingCanvasNames,
}: Pick<
  CreateIntakeAutomationInput,
  "factoryId" | "sourceId" | "createCanvas" | "existingCanvasNames"
>): Promise<string> {
  const preferredName = intakeAutomationName(sourceId);
  const taken = new Set([...(existingCanvasNames ?? [])].map((name) => name.trim()).filter((name) => Boolean(name)));
  let name = uniqueCanvasName(preferredName, taken);

  for (let attempt = 0; attempt < MAX_NAME_RETRY_ATTEMPTS; attempt++) {
    try {
      const response = await createCanvas({
        name,
        description: intakeAutomationDescription(sourceId),
        factoryId,
        method: "template",
      });
      const canvasId = response.data?.canvas?.metadata?.id;
      if (!canvasId) {
        throw new Error("Failed to create intake automation");
      }
      return canvasId;
    } catch (error) {
      if (!isCanvasNameAlreadyExistsError(error)) {
        throw error;
      }
      taken.add(name);
      name = uniqueCanvasName(preferredName, taken);
    }
  }

  throw new Error("Failed to create intake automation");
}

async function stageIntakeCanvas(canvasId: string, canvasYaml: string): Promise<unknown> {
  return canvasesPutCanvasStaging(
    withOrganizationHeader({
      path: { canvasId },
      body: {
        operations: [
          {
            path: CANVAS_YAML_PATH,
            content: encodeRepositoryFileContent(canvasYaml),
          },
        ],
      },
    }),
  );
}

async function commitIntakeCanvas(canvasId: string): Promise<unknown> {
  return canvasesCommitCanvasStaging(
    withOrganizationHeader({
      path: { canvasId },
      body: { commitMessage: "Add intake automation" },
    }),
  );
}

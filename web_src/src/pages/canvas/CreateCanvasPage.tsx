import { OrganizationMenuButton } from "@/components/OrganizationMenuButton";
import { usePageTitle } from "@/hooks/usePageTitle";
import { useState } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Field, Label } from "../../components/Fieldset/fieldset";
import { Heading } from "../../components/Heading/heading";
import { Input } from "../../components/Input/input";
import { Text } from "../../components/Text/text";
import { Textarea } from "../../components/ui/textarea";
import { Button } from "../../components/ui/button";
import { useCreateWorkflow } from "../../hooks/useWorkflowData";
import { showErrorToast } from "../../utils/toast";

const MAX_CANVAS_NAME_LENGTH = 50;
const MAX_CANVAS_DESCRIPTION_LENGTH = 200;

export function CreateCanvasPage() {
  usePageTitle(["New Canvas"]);
  const { organizationId } = useParams<{ organizationId: string }>();
  const navigate = useNavigate();

  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [nameError, setNameError] = useState("");

  const createMutation = useCreateWorkflow(organizationId || "");

  const handleSubmit = async () => {
    setNameError("");

    if (!name.trim()) {
      setNameError("Name is required");
      return;
    }

    if (name.trim().length > MAX_CANVAS_NAME_LENGTH) {
      setNameError(`Name must be ${MAX_CANVAS_NAME_LENGTH} characters or less`);
      return;
    }

    if (!organizationId) {
      showErrorToast("Organization ID is missing");
      return;
    }

    try {
      const result = await createMutation.mutateAsync({
        name: name.trim(),
        description: description.trim() || undefined,
      });

      if (result?.data?.workflow?.metadata?.id) {
        navigate(`/${organizationId}/workflows/${result.data.workflow.metadata.id}`);
      }
    } catch (error) {
      console.error("Error creating canvas:", error);
      const errorMessage = (error as Error)?.message || error?.toString() || "Failed to create canvas";

      showErrorToast(errorMessage);

      if (errorMessage.toLowerCase().includes("already") || errorMessage.toLowerCase().includes("exists")) {
        setNameError("A canvas with this name already exists");
      }
    }
  };

  const handleCancel = () => {
    navigate(`/${organizationId}`);
  };

  return (
    <div className="min-h-screen flex flex-col bg-gray-50 dark:bg-gray-900">
      <header className="bg-white border-b border-border px-4 h-12 flex items-center">
        <OrganizationMenuButton organizationId={organizationId || ""} />
      </header>
      <main className="w-full h-full flex flex-column flex-grow-1">
        <div className="w-full flex-grow-1">
          <div className="p-8 max-w-lg mx-auto">
            <div className="mb-6">
              <Heading level={2} className="!text-xl mb-1">
                New Canvas
              </Heading>
              <Text className="text-gray-800 dark:text-gray-400">
                Create a new canvas to orchestrate your DevOps work.
              </Text>
            </div>

            <div className="bg-white dark:bg-gray-950 rounded-lg border border-gray-300 dark:border-gray-800 p-6 space-y-6">
              <Field>
                <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Canvas name *</Label>
                <Input
                  data-testid="canvas-name-input"
                  type="text"
                  autoComplete="off"
                  value={name}
                  onChange={(e) => {
                    if (e.target.value.length <= MAX_CANVAS_NAME_LENGTH) {
                      setName(e.target.value);
                    }
                    if (nameError) {
                      setNameError("");
                    }
                  }}
                  placeholder=""
                  className={`w-full ${nameError ? "border-red-500" : ""}`}
                  autoFocus
                  maxLength={MAX_CANVAS_NAME_LENGTH}
                  onKeyDown={(e) => {
                    if (e.key === "Enter" && !e.shiftKey) {
                      e.preventDefault();
                      handleSubmit();
                    }
                  }}
                />
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  {name.length}/{MAX_CANVAS_NAME_LENGTH} characters
                </div>
                {nameError && <div className="text-xs text-red-600 mt-1">{nameError}</div>}
              </Field>

              <Field>
                <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Description</Label>
                <Textarea
                  value={description}
                  onChange={(e) => {
                    if (e.target.value.length <= MAX_CANVAS_DESCRIPTION_LENGTH) {
                      setDescription(e.target.value);
                    }
                  }}
                  placeholder="Describe what it does (optional)"
                  rows={3}
                  className="w-full"
                  maxLength={MAX_CANVAS_DESCRIPTION_LENGTH}
                />
                <div className="text-xs text-gray-500 dark:text-gray-400 mt-1">
                  {description.length}/{MAX_CANVAS_DESCRIPTION_LENGTH} characters
                </div>
              </Field>

              <div className="flex justify-start gap-3">
                <Button
                  onClick={handleSubmit}
                  disabled={!name.trim() || createMutation.isPending || !!nameError}
                  data-testid="create-canvas-button"
                >
                  {createMutation.isPending ? "Creating Canvas..." : "Create Canvas"}
                </Button>
                <Button variant="outline" onClick={handleCancel}>
                  Cancel
                </Button>
              </div>
            </div>
          </div>

          <div className="p-8 max-w-5xl mx-auto">
            <Heading level={3} className="!text-sm mb-4">
              Start from an example
            </Heading>
            <div className="grid grid-cols-3 gap-6">
              <ExampleCard
                title="Example 1"
                description="This is the first example canvas template"
                onClick={() => {}}
              />
              <ExampleCard
                title="Example 2"
                description="This is the second example canvas template"
                onClick={() => {}}
              />
              <ExampleCard
                title="Example 3"
                description="This is the third example canvas template"
                onClick={() => {}}
              />
            </div>
          </div>
        </div>
      </main>
    </div>
  );
}

interface ExampleCardProps {
  title: string;
  description: string;
  onClick: () => void;
}

function ExampleCard({ title, description, onClick }: ExampleCardProps) {
  return (
    <div
      onClick={onClick}
      className="bg-white dark:bg-gray-950 rounded-lg border border-gray-300 dark:border-gray-800 p-4 cursor-pointer hover:shadow-md transition-shadow"
    >
      <div className="h-32 w-full rounded mb-3"></div>
      <h4 className="text-sm font-medium text-gray-800 dark:text-white mb-1">{title}</h4>
      <p className="text-[13px] text-gray-500 dark:text-gray-400">{description}</p>
    </div>
  );
}

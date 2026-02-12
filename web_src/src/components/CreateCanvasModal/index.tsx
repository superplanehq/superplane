import { useCallback, useEffect, useRef, useState } from "react";
import { Upload } from "lucide-react";
import { showErrorToast } from "../../utils/toast";
import { parseCanvasYaml, readFileAsText, type ParsedCanvas } from "../../utils/parseCanvasYaml";
import type { ComponentsNode, ComponentsEdge } from "@/api-client";
import { Dialog, DialogActions, DialogBody, DialogDescription, DialogTitle } from "../Dialog/dialog";
import { Field, Label } from "../Fieldset/fieldset";
import { Icon } from "../Icon";
import { Input } from "../Input/input";
import { Textarea } from "../ui/textarea";
import { Button } from "../ui/button";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../ui/tabs";

export interface CreateCanvasSubmitData {
  name: string;
  description?: string;
  templateId?: string;
  nodes?: ComponentsNode[];
  edges?: ComponentsEdge[];
}

interface CreateCanvasModalProps {
  isOpen: boolean;
  onClose: () => void;
  onSubmit: (data: CreateCanvasSubmitData) => Promise<void>;
  isLoading?: boolean;
  initialData?: { name: string; description?: string };
  templates?: { id: string; name: string; description?: string }[];
  defaultTemplateId?: string;
  mode?: "create" | "edit";
  fromTemplate?: boolean;
}

const MAX_CANVAS_NAME_LENGTH = 50;
const MAX_CANVAS_DESCRIPTION_LENGTH = 200;

export function CreateCanvasModal({
  isOpen,
  onClose,
  onSubmit,
  isLoading = false,
  initialData,
  defaultTemplateId,
  mode = "create",
  fromTemplate = false,
}: CreateCanvasModalProps) {
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [nameError, setNameError] = useState("");
  const [templateId, setTemplateId] = useState("");

  const [activeTab, setActiveTab] = useState<string>("manual");
  const [yamlText, setYamlText] = useState("");
  const [yamlError, setYamlError] = useState("");
  const [importedSpec, setImportedSpec] = useState<{ nodes: ComponentsNode[]; edges: ComponentsEdge[] } | null>(null);
  const fileInputRef = useRef<HTMLInputElement>(null);

  useEffect(() => {
    if (isOpen) {
      setName(initialData?.name ?? "");
      setDescription(initialData?.description ?? "");
      setNameError("");
      setActiveTab("manual");
      setYamlText("");
      setYamlError("");
      setImportedSpec(null);
    }
    if (isOpen && mode === "create") {
      setTemplateId(defaultTemplateId || "");
    }
    if (isOpen && mode !== "create") {
      setTemplateId("");
    }
  }, [isOpen, initialData?.name, initialData?.description, defaultTemplateId, mode]);

  const handleClose = () => {
    setName("");
    setDescription("");
    setNameError("");
    setTemplateId("");
    setYamlText("");
    setYamlError("");
    setImportedSpec(null);
    onClose();
  };

  const applyParsedYaml = useCallback((parsed: ParsedCanvas) => {
    setName(parsed.name.slice(0, MAX_CANVAS_NAME_LENGTH));
    setDescription((parsed.description ?? "").slice(0, MAX_CANVAS_DESCRIPTION_LENGTH));
    setImportedSpec({ nodes: parsed.nodes, edges: parsed.edges });
    setYamlError("");
    setNameError("");
  }, []);

  const handleYamlParse = useCallback(() => {
    if (!yamlText.trim()) {
      setYamlError("Paste or upload a YAML file first.");
      setImportedSpec(null);
      return;
    }

    try {
      const parsed = parseCanvasYaml(yamlText);
      applyParsedYaml(parsed);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setYamlError(message);
      setImportedSpec(null);
    }
  }, [yamlText, applyParsedYaml]);

  const handleFileUpload = useCallback(
    async (event: React.ChangeEvent<HTMLInputElement>) => {
      const file = event.target.files?.[0];
      if (!file) return;

      try {
        const text = await readFileAsText(file);
        setYamlText(text);

        const parsed = parseCanvasYaml(text);
        applyParsedYaml(parsed);
      } catch (err) {
        const message = err instanceof Error ? err.message : String(err);
        setYamlError(message);
        setImportedSpec(null);
      }

      // Reset file input so re-uploading the same file triggers onChange
      if (fileInputRef.current) {
        fileInputRef.current.value = "";
      }
    },
    [applyParsedYaml],
  );

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

    try {
      await onSubmit({
        name: name.trim(),
        description: description.trim() || undefined,
        templateId: templateId || undefined,
        nodes: importedSpec?.nodes,
        edges: importedSpec?.edges,
      });

      // Reset form and close modal
      setName("");
      setDescription("");
      setNameError("");
      setTemplateId("");
      setYamlText("");
      setYamlError("");
      setImportedSpec(null);
      onClose();
    } catch (error) {
      const errorMessage = (error as Error)?.message || error?.toString() || "Failed to create canvas";

      showErrorToast(errorMessage);

      if (errorMessage.toLowerCase().includes("already") || errorMessage.toLowerCase().includes("exists")) {
        setNameError("A canvas with this name already exists");
      }
    }
  };

  const showYamlTab = mode === "create" && !fromTemplate;

  return (
    <Dialog open={isOpen} onClose={handleClose} size="lg" className="text-left relative">
      <DialogTitle>
        {fromTemplate ? "New Canvas from template" : mode === "edit" ? "Edit Canvas" : "New Canvas"}
      </DialogTitle>
      <DialogDescription className="text-sm !text-[var(--color-gray-800)]">
        {fromTemplate
          ? "Create a canvas from this template. Give it a name and optional description to get started."
          : mode === "edit"
            ? "Update the canvas details to keep things clear for your teammates."
            : "Create a new canvas or import one from a YAML file."}
      </DialogDescription>
      <button onClick={handleClose} className="absolute top-4 right-4">
        <Icon name="close" size="sm" />
      </button>

      <DialogBody>
        {showYamlTab ? (
          <Tabs value={activeTab} onValueChange={setActiveTab}>
            <TabsList className="mb-4">
              <TabsTrigger value="manual">Create manually</TabsTrigger>
              <TabsTrigger value="yaml">Import from YAML</TabsTrigger>
            </TabsList>

            <TabsContent value="manual">
              <CanvasFormFields
                name={name}
                description={description}
                nameError={nameError}
                onNameChange={setName}
                onDescriptionChange={setDescription}
                onNameErrorChange={setNameError}
              />
            </TabsContent>

            <TabsContent value="yaml">
              <div className="space-y-4">
                <div>
                  <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">
                    YAML content
                  </Label>
                  <Textarea
                    data-testid="yaml-import-textarea"
                    value={yamlText}
                    onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => {
                      setYamlText(e.target.value);
                      if (yamlError) setYamlError("");
                    }}
                    placeholder={`metadata:\n  name: My Canvas\n  description: Optional description\nspec:\n  nodes: []\n  edges: []`}
                    rows={10}
                    className="w-full font-mono text-sm"
                  />
                  {yamlError && <div className="text-xs text-red-600 mt-1">{yamlError}</div>}
                  {importedSpec && (
                    <div className="text-xs text-green-700 mt-1">
                      Parsed successfully: {importedSpec.nodes.length} node(s), {importedSpec.edges.length} edge(s).
                    </div>
                  )}
                </div>

                <div className="flex items-center gap-3">
                  <Button type="button" variant="outline" size="sm" onClick={handleYamlParse}>
                    Parse YAML
                  </Button>
                  <input
                    ref={fileInputRef}
                    type="file"
                    accept=".yaml,.yml"
                    onChange={handleFileUpload}
                    className="hidden"
                    data-testid="yaml-file-input"
                  />
                  <Button
                    type="button"
                    variant="outline"
                    size="sm"
                    onClick={() => fileInputRef.current?.click()}
                    className="flex items-center gap-1.5"
                  >
                    <Upload className="h-3.5 w-3.5" />
                    Upload file
                  </Button>
                </div>

                
                {importedSpec && (
                  <div className="border-t border-gray-200 dark:border-gray-700 pt-4">
                    <CanvasFormFields
                      name={name}
                      description={description}
                      nameError={nameError}
                      onNameChange={setName}
                      onDescriptionChange={setDescription}
                      onNameErrorChange={setNameError}
                    />
                  </div>
                )}
              </div>
            </TabsContent>
          </Tabs>
        ) : (
          <CanvasFormFields
            name={name}
            description={description}
            nameError={nameError}
            onNameChange={setName}
            onDescriptionChange={setDescription}
            onNameErrorChange={setNameError}
          />
        )}
      </DialogBody>

      <DialogActions>
        <Button
          onClick={handleSubmit}
          disabled={!name.trim() || isLoading || !!nameError || (activeTab === "yaml" && !importedSpec)}
          className="flex items-center gap-2"
          data-testid="create-canvas-submit"
        >
          {mode === "edit"
            ? isLoading
              ? "Saving..."
              : "Save changes"
            : fromTemplate
              ? isLoading
                ? "Creating Canvas"
                : "Create Canvas"
              : isLoading
                ? "Creating canvas..."
                : "Create canvas"}
        </Button>
      </DialogActions>
    </Dialog>
  );
}

function CanvasFormFields({
  name,
  description,
  nameError,
  onNameChange,
  onDescriptionChange,
  onNameErrorChange,
}: {
  name: string;
  description: string;
  nameError: string;
  onNameChange: (value: string) => void;
  onDescriptionChange: (value: string) => void;
  onNameErrorChange: (value: string) => void;
}) {
  return (
    <div className="space-y-6">
      <Field>
        <Label className="block text-sm font-medium text-gray-700 dark:text-gray-300 mb-2">Canvas name *</Label>
        <Input
          data-testid="canvas-name-input"
          type="text"
          autoComplete="off"
          value={name}
          onChange={(e: React.ChangeEvent<HTMLInputElement>) => {
            if (e.target.value.length <= MAX_CANVAS_NAME_LENGTH) {
              onNameChange(e.target.value);
            }
            if (nameError) {
              onNameErrorChange("");
            }
          }}
          placeholder=""
          className={`w-full ${nameError ? "border-red-500" : ""}`}
          autoFocus
          maxLength={MAX_CANVAS_NAME_LENGTH}
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
          onChange={(e: React.ChangeEvent<HTMLTextAreaElement>) => {
            if (e.target.value.length <= MAX_CANVAS_DESCRIPTION_LENGTH) {
              onDescriptionChange(e.target.value);
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
    </div>
  );
}

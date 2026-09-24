import { useEffect, useState } from "react";

import type { FactoriesFactoryAgentResource } from "@/api-client";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { LoadingButton } from "@/components/ui/loading-button";
import { Textarea } from "@/components/ui/textarea";

import { AGENT_RESOURCES_COPY } from "./agentResourceCopy";

const NAME_PATTERN = /^[a-z][a-z0-9-]{0,62}$/;
const RESERVED_NAME = "superplane";

export type AgentResourceSkillDraft = {
  name: string;
  markdown: string;
};

function validateName(name: string): string {
  if (!name) {
    return AGENT_RESOURCES_COPY.nameRequired;
  }
  if (name === RESERVED_NAME) {
    return AGENT_RESOURCES_COPY.nameReserved;
  }
  if (!NAME_PATTERN.test(name)) {
    return AGENT_RESOURCES_COPY.nameInvalid;
  }
  return "";
}

function validateMarkdown(markdown: string): string {
  if (!markdown) {
    return AGENT_RESOURCES_COPY.markdownRequired;
  }
  return "";
}

export function AgentResourceSkillDialog({
  open,
  resource,
  isSaving,
  onClose,
  onSave,
}: {
  open: boolean;
  resource?: FactoriesFactoryAgentResource;
  isSaving: boolean;
  onClose: () => void;
  onSave: (draft: AgentResourceSkillDraft) => Promise<void>;
}) {
  const isEdit = Boolean(resource);
  const [name, setName] = useState("");
  const [markdown, setMarkdown] = useState("");
  const [nameError, setNameError] = useState("");
  const [markdownError, setMarkdownError] = useState("");

  useEffect(() => {
    if (!open) {
      return;
    }
    setName(resource?.name ?? "");
    setMarkdown(resource?.markdown ?? "");
    setNameError("");
    setMarkdownError("");
  }, [open, resource]);

  const handleSave = async () => {
    const trimmedName = name.trim().toLowerCase();
    const trimmedMarkdown = markdown.trim();
    const nextNameError = validateName(trimmedName);
    const nextMarkdownError = validateMarkdown(trimmedMarkdown);
    setNameError(nextNameError);
    setMarkdownError(nextMarkdownError);
    if (nextNameError || nextMarkdownError) {
      return;
    }
    await onSave({
      name: trimmedName,
      markdown: trimmedMarkdown,
    });
  };

  return (
    <Dialog open={open} onOpenChange={(nextOpen) => !nextOpen && !isSaving && onClose()}>
      <DialogContent data-testid="agent-resource-skill-dialog">
        <DialogHeader>
          <DialogTitle>{isEdit ? AGENT_RESOURCES_COPY.editSkill : AGENT_RESOURCES_COPY.addSkill}</DialogTitle>
          <DialogDescription>{AGENT_RESOURCES_COPY.skillDialogDescription}</DialogDescription>
        </DialogHeader>
        <div className="flex flex-col gap-4">
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="agent-resource-skill-name">{AGENT_RESOURCES_COPY.nameLabel}</Label>
            <Input
              id="agent-resource-skill-name"
              data-testid="agent-resource-skill-name"
              value={name}
              onChange={(event) => setName(event.target.value)}
              placeholder="review-copy"
              autoComplete="off"
            />
            <p className="text-[12px] text-muted-foreground">{AGENT_RESOURCES_COPY.nameHelper}</p>
            {nameError ? <p className="text-[12px] text-destructive">{nameError}</p> : null}
          </div>
          <div className="flex flex-col gap-1.5">
            <Label htmlFor="agent-resource-skill-markdown">{AGENT_RESOURCES_COPY.markdownLabel}</Label>
            <Textarea
              id="agent-resource-skill-markdown"
              data-testid="agent-resource-skill-markdown"
              className="min-h-48 font-mono"
              value={markdown}
              onChange={(event) => setMarkdown(event.target.value)}
              placeholder={"---\nname: review-copy\ndescription: Write STE UI copy.\n---\n"}
              autoComplete="off"
            />
            <p className="text-[12px] text-muted-foreground">{AGENT_RESOURCES_COPY.markdownHelper}</p>
            <p className="text-[12px] text-muted-foreground">{AGENT_RESOURCES_COPY.skillExtraFilesNote}</p>
            {markdownError ? <p className="text-[12px] text-destructive">{markdownError}</p> : null}
          </div>
        </div>
        <DialogFooter>
          <Button type="button" variant="outline" onClick={onClose} disabled={isSaving}>
            {AGENT_RESOURCES_COPY.cancel}
          </Button>
          <LoadingButton
            type="button"
            onClick={() => void handleSave()}
            loading={isSaving}
            data-testid="agent-resource-skill-save"
          >
            {isEdit ? AGENT_RESOURCES_COPY.saveSkill : AGENT_RESOURCES_COPY.addSkill}
          </LoadingButton>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

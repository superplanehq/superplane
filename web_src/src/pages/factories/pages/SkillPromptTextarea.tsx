import { useEffect, useLayoutEffect, useId, useRef, useState, type KeyboardEvent, type RefObject } from "react";
import { createPortal } from "react-dom";
import { Link } from "@/components/Link/link";
import { HoverCard, HoverCardContent, HoverCardTrigger } from "@/components/ui/hover-card";
import { BookOpen } from "lucide-react";
import type { FactoriesFactoryAgentResource } from "@/api-client";
import { Button } from "@/components/ui/button";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { Textarea } from "@/components/ui/textarea";
import { cn } from "@/lib/utils";
import { skillDisplayTitle, skillListDescription } from "./settings/skillFrontmatter";

const PROMPT_TEXTAREA_CLASS =
  "max-h-72 min-h-24 resize-y overflow-auto rounded-lg bg-card px-3 py-2.5 text-[13px] leading-relaxed shadow-none";

/** Preserve plain prompt text, using the same transparent-textarea backdrop as expression inputs. */
export function SkillPromptTextarea({
  value,
  onChange,
  skills,
  label,
  testId,
  skillHref,
}: {
  skillHref?: (skill: FactoriesFactoryAgentResource) => string | undefined;
  value: string;
  onChange: (value: string) => void;
  skills: FactoriesFactoryAgentResource[];
  label: string;
  testId: string;
}) {
  const textarea = useRef<HTMLTextAreaElement>(null);
  const backdrop = useRef<HTMLDivElement>(null);
  const listId = useId();
  const {
    matches,
    open,
    active,
    rect,
    insert,
    handleKeyDown,
    updateCursor,
    setFocused,
    setCursor,
    setSelected,
    setDismissed,
    focusToken,
  } = useSkillPromptCompletion({ value, onChange, skills, textarea, backdrop, listId });

  return (
    <>
      <div className="relative rounded-lg bg-card" data-testid={`${testId}-skills`}>
        <div
          ref={backdrop}
          aria-hidden="true"
          className={cn(
            PROMPT_TEXTAREA_CLASS,
            "pointer-events-none absolute inset-0 z-10 bg-transparent overflow-hidden whitespace-pre-wrap wrap-anywhere border border-transparent text-foreground",
          )}
        >
          <SkillPromptTokens value={value} skills={skills} onSelect={focusToken} skillHref={skillHref} />
          {"\n"}
        </div>
        <Textarea
          ref={textarea}
          value={value}
          aria-label={label}
          data-testid={testId}
          placeholder="Tell the agent what to do in this step."
          className={cn(
            PROMPT_TEXTAREA_CLASS,
            "relative bg-transparent dark:bg-transparent text-transparent dark:text-transparent caret-foreground selection:bg-primary/20",
          )}
          aria-autocomplete="list"
          aria-controls={open ? listId : undefined}
          aria-activedescendant={open && matches.length ? `${listId}-${active}` : undefined}
          onChange={(event) => {
            onChange(event.target.value);
            setCursor(event.target.selectionStart);
            setSelected(0);
            setDismissed(false);
          }}
          onSelect={updateCursor}
          onFocus={() => {
            setFocused(true);
            updateCursor();
          }}
          onBlur={() => setFocused(false)}
          onKeyDown={handleKeyDown}
          onScroll={(event) => {
            if (backdrop.current) {
              backdrop.current.scrollTop = event.currentTarget.scrollTop;
              backdrop.current.scrollLeft = event.currentTarget.scrollLeft;
            }
            setDismissed(true);
          }}
        />
        {open && rect
          ? createPortal(
              <div
                id={listId}
                role="listbox"
                aria-label="Available skills"
                className="fixed z-[9999] max-h-56 overflow-auto rounded-lg border border-border bg-popover p-1 text-popover-foreground shadow-lg"
                style={{
                  left: Math.max(8, Math.min(rect.left, window.innerWidth - 328)),
                  top: Math.max(8, Math.min(rect.bottom + 4, window.innerHeight - 232)),
                  width: Math.min(320, window.innerWidth - 16),
                }}
              >
                <div className="px-2 py-1 text-xs text-muted-foreground">Skills</div>
                {matches.length ? (
                  matches.map((skill, index) => (
                    <SkillSuggestion
                      key={skill.id ?? skill.name}
                      skill={skill}
                      id={`${listId}-${index}`}
                      selected={index === active}
                      onSelect={() => insert(skill)}
                      onHover={() => setSelected(index)}
                    />
                  ))
                ) : (
                  <p className="px-2 py-2 text-xs text-muted-foreground">
                    {skills.length ? "No matching skills." : "No skills available for this agent."}
                  </p>
                )}
              </div>,
              document.body,
            )
          : null}
      </div>
      <PromptSkillDetails value={value} skills={skills} skillHref={skillHref} />
    </>
  );
}

function useSkillPromptCompletion({
  value,
  onChange,
  skills,
  textarea,
  backdrop,
  listId,
}: {
  value: string;
  onChange: (value: string) => void;
  skills: FactoriesFactoryAgentResource[];
  textarea: RefObject<HTMLTextAreaElement | null>;
  backdrop: RefObject<HTMLDivElement | null>;
  listId: string;
}) {
  const [cursor, setCursor] = useState(0);
  const [focused, setFocused] = useState(false);
  const [dismissed, setDismissed] = useState(false);
  const [selected, setSelected] = useState(0);
  const command = skillCommandAtCursor(value, cursor);
  const matches = skills.filter((skill) => skill.name?.toLowerCase().startsWith(command?.query.toLowerCase() ?? ""));
  const complete = skills.some((skill) => skill.name === command?.query);
  const open = focused && !dismissed && Boolean(command) && !complete;
  const active = Math.min(selected, Math.max(0, matches.length - 1));
  const rect = textarea.current?.getBoundingClientRect();

  useLayoutEffect(() => {
    const field = textarea.current;
    if (!field || !backdrop.current) return;
    const sync = () => {
      if (!backdrop.current) return;
      backdrop.current.style.width = `${field.clientWidth + 2}px`;
      backdrop.current.scrollTop = field.scrollTop;
    };
    sync();
    const observer = new ResizeObserver(sync);
    observer.observe(field);
    return () => observer.disconnect();
  }, [value, textarea, backdrop]);
  useEffect(() => {
    if (open) document.getElementById(`${listId}-${active}`)?.scrollIntoView?.({ block: "nearest" });
  }, [open, active, listId]);

  const updateCursor = () => {
    setCursor(textarea.current?.selectionStart ?? 0);
    setDismissed(false);
  };
  const insert = (skill: FactoriesFactoryAgentResource) => {
    if (!command || !skill.name) return;
    const token = `/${skill.name}`;
    const tail = value.slice(command.end);
    const separator = /^\s/.test(tail) ? "" : " ";
    const next = value.slice(0, command.start) + token + separator + tail;
    onChange(next);
    setDismissed(true);
    requestAnimationFrame(() => {
      const position = command.start + token.length + (separator.length || 1);
      textarea.current?.focus();
      textarea.current?.setSelectionRange(position, position);
      setCursor(position);
    });
  };
  const focusToken = (position: number) => {
    if (!/[ \t]/.test(value.charAt(position))) {
      onChange(value.slice(0, position) + " " + value.slice(position));
    }
    setDismissed(true);
    requestAnimationFrame(() => {
      const afterToken = position + 1;
      textarea.current?.focus();
      textarea.current?.setSelectionRange(afterToken, afterToken);
      setCursor(afterToken);
    });
  };
  const handleKeyDown = (event: KeyboardEvent<HTMLTextAreaElement>) => {
    if (!open || event.nativeEvent.isComposing) return;
    if (event.key === "Escape") {
      event.preventDefault();
      event.stopPropagation();
      setDismissed(true);
      return;
    }
    if (matches.length === 0) return;
    if (event.key === "ArrowDown" || event.key === "ArrowUp") {
      event.preventDefault();
      setSelected((active + (event.key === "ArrowDown" ? 1 : -1) + matches.length) % matches.length);
    } else if (event.key === "Enter" || event.key === "Tab") {
      event.preventDefault();
      insert(matches[active]);
    }
  };

  return {
    matches,
    open,
    active,
    rect,
    insert,
    handleKeyDown,
    updateCursor,
    setFocused,
    setCursor,
    setSelected,
    setDismissed,
    focusToken,
  };
}

function SkillSuggestion({
  skill,
  id,
  selected,
  onSelect,
  onHover,
}: {
  skill: FactoriesFactoryAgentResource;
  id: string;
  selected: boolean;
  onSelect: () => void;
  onHover: () => void;
}) {
  const description = skillListDescription(skill);
  return (
    <div
      id={id}
      role="option"
      aria-selected={selected}
      className={cn("flex cursor-pointer items-start gap-2 rounded-md px-2 py-2 text-[13px]", selected && "bg-accent")}
      onMouseDown={(event) => event.preventDefault()}
      onMouseEnter={onHover}
      onClick={onSelect}
    >
      <BookOpen className="mt-0.5 size-3.5 shrink-0 text-muted-foreground" aria-hidden />
      <div className="min-w-0">
        <div>/{skill.name}</div>
        {description ? <p className="truncate text-xs text-muted-foreground">{description}</p> : null}
      </div>
    </div>
  );
}

function skillCommandAtCursor(value: string, cursor: number) {
  const match = /(^|[\s(])\/([a-zA-Z0-9_-]*)$/.exec(value.slice(0, cursor));
  if (!match) return undefined;
  const end = cursor + (/^[a-zA-Z0-9_-]*/.exec(value.slice(cursor))?.[0].length ?? 0);
  if (end < value.length && !/[\s.,!?;:)]/.test(value[end])) return undefined;
  return { query: match[2], start: match.index + match[1].length, end };
}

function SkillPromptTokens({
  value,
  skills,
  onSelect,
  skillHref,
}: {
  skillHref?: (skill: FactoriesFactoryAgentResource) => string | undefined;
  value: string;
  skills: FactoriesFactoryAgentResource[];
  onSelect: (position: number) => void;
}) {
  const parts: React.ReactNode[] = [];
  let position = 0;
  for (const { skill, start, end } of skillRanges(value, skills)) {
    parts.push(value.slice(position, start));
    parts.push(
      <SkillToken
        key={start}
        skill={skill}
        command={value.slice(start, end)}
        href={skillHref?.(skill)}
        onSelect={() => onSelect(end)}
      />,
    );
    position = end;
  }
  parts.push(value.slice(position));
  return <>{parts}</>;
}

function SkillToken({
  skill,
  command,
  href,
  onSelect,
}: {
  skill: FactoriesFactoryAgentResource;
  command: string;
  href?: string;
  onSelect: () => void;
}) {
  return (
    <HoverCard openDelay={250} closeDelay={150}>
      <HoverCardTrigger asChild>
        <Button
          variant="ghost"
          type="button"
          tabIndex={-1}
          data-skill-token={skill.name}
          data-skill-id={skill.id}
          className="pointer-events-auto inline h-auto whitespace-pre-wrap cursor-text rounded-md border-0 text-[13px] leading-[inherit] font-normal shadow-none hover:bg-violet-500/15 hover:text-violet-700 bg-violet-500/15 -mx-0.5 px-0.5 py-0 text-violet-700 ring-1 ring-inset ring-violet-500/30 dark:bg-violet-500/20 dark:text-violet-300 dark:ring-violet-500/50 dark:hover:bg-violet-500/20 dark:hover:text-violet-300"
          onMouseDown={(event) => {
            event.preventDefault();
            onSelect();
          }}
        >
          {command}
        </Button>
      </HoverCardTrigger>
      <HoverCardContent
        align="start"
        className="pointer-events-auto z-[10000] w-72 space-y-2 p-3"
        data-testid="prompt-skill-hover-card"
      >
        <SkillDetails skill={skill} href={href} />
      </HoverCardContent>
    </HoverCard>
  );
}

function skillRanges(value: string, skills: FactoriesFactoryAgentResource[]) {
  const ranges: { skill: FactoriesFactoryAgentResource; start: number; end: number }[] = [];
  for (const match of value.matchAll(/(^|[\s(])\/([a-zA-Z0-9_-]+)(?=$|[\s.,!?;:)])/g)) {
    const skill = skills.find((resource) => resource.name === match[2]);
    if (!skill) continue;
    const start = match.index + match[1].length;
    ranges.push({ skill, start, end: start + match[2].length + 1 });
  }
  return ranges;
}

function PromptSkillDetails({
  value,
  skills,
  skillHref,
}: {
  value: string;
  skills: FactoriesFactoryAgentResource[];
  skillHref?: (skill: FactoriesFactoryAgentResource) => string | undefined;
}) {
  const referenced = [...new Map(skillRanges(value, skills).map(({ skill }) => [skill.name, skill])).values()];
  if (referenced.length === 0) return null;
  return (
    <div className="flex gap-1" aria-label="Skill details">
      {referenced.map((skill) => (
        <Popover key={skill.id ?? skill.name}>
          <PopoverTrigger asChild>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              className="sr-only focus:not-sr-only data-[state=open]:not-sr-only"
              aria-label={`Skill details: ${skillDisplayTitle(skill)}`}
            >
              <BookOpen className="size-3.5" aria-hidden />
              {skillDisplayTitle(skill)}
            </Button>
          </PopoverTrigger>
          <PopoverContent
            align="start"
            className="z-[10000] space-y-2 p-3"
            aria-label={`Skill details: ${skillDisplayTitle(skill)}`}
          >
            <SkillDetails skill={skill} href={skillHref?.(skill)} />
          </PopoverContent>
        </Popover>
      ))}
    </div>
  );
}

function SkillDetails({ skill, href }: { skill: FactoriesFactoryAgentResource; href?: string }) {
  const description = skillListDescription(skill);
  return (
    <>
      <h4 className="text-[13px] font-semibold">{skillDisplayTitle(skill)}</h4>
      {description ? <p className="text-[13px] leading-5 text-muted-foreground">{description}</p> : null}
      {href ? (
        <Link
          href={href}
          target="_blank"
          rel="noopener noreferrer"
          className="inline-block text-[13px] font-medium underline underline-offset-2"
        >
          Open skill <span aria-hidden="true">↗</span>
        </Link>
      ) : null}
    </>
  );
}

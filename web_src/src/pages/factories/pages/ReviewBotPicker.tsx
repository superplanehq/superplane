import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { cn } from "@/lib/utils";
import { Check, Loader2 } from "lucide-react";
import { useMemo, useState } from "react";

import { PR_FEEDBACK_SETTINGS_COPY } from "./prFeedbackSettingsModel";
import { normalizeReviewBotLogin, reviewBotRows, type ReviewBotOption } from "./reviewBotPickerModel";

export function ReviewBotPicker({
  selected,
  catalog,
  loading,
  loadError,
  supportsReviewBots = true,
  onToggle,
  onAdd,
}: {
  selected: string[];
  catalog: ReviewBotOption[];
  loading?: boolean;
  loadError?: boolean;
  supportsReviewBots?: boolean;
  onToggle: (login: string) => void;
  onAdd: (login: string) => boolean;
}) {
  if (!supportsReviewBots) {
    return <ReviewBotNameEntry selected={selected} onToggle={onToggle} onAdd={onAdd} />;
  }

  return (
    <ReviewBotCatalogPicker
      selected={selected}
      catalog={catalog}
      loading={loading}
      loadError={loadError}
      onToggle={onToggle}
      onAdd={onAdd}
    />
  );
}

function ReviewBotNameEntry({
  selected,
  onToggle,
  onAdd,
}: {
  selected: string[];
  onToggle: (login: string) => void;
  onAdd: (login: string) => boolean;
}) {
  const [draft, setDraft] = useState("");

  const submitDraft = () => {
    if (!onAdd(draft)) {
      return;
    }
    setDraft("");
  };

  return (
    <section className="space-y-3">
      <p className="workspace-body-text text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.wizardBotsNameInstruction}</p>
      <div className="space-y-1.5">
        <Label htmlFor="discussion-setup-bot-manual">{PR_FEEDBACK_SETTINGS_COPY.wizardBotsNameLabel}</Label>
        <ReviewBotManualAdd
          draft={draft}
          label={PR_FEEDBACK_SETTINGS_COPY.wizardBotsNameLabel}
          placeholder={PR_FEEDBACK_SETTINGS_COPY.wizardBotsNamePlaceholder}
          onDraftChange={setDraft}
          onSubmit={submitDraft}
        />
      </div>
      <ReviewBotNameList selected={selected} onToggle={onToggle} />
    </section>
  );
}

function ReviewBotNameList({ selected, onToggle }: { selected: string[]; onToggle: (login: string) => void }) {
  const names = selected.map((login) => login.trim()).filter((login) => login.length > 0);
  if (names.length === 0) {
    return null;
  }

  return (
    <ul className="divide-y divide-border rounded-lg border border-border" data-testid="discussion-setup-bot-names">
      {names.map((login) => (
        <li key={login.toLowerCase()} className="flex items-center justify-between gap-3 px-3 py-2">
          <span className="min-w-0 truncate text-[13px] font-medium">{login}</span>
          <Button
            type="button"
            variant="ghost"
            size="sm"
            onClick={() => onToggle(login)}
            aria-label={`${PR_FEEDBACK_SETTINGS_COPY.wizardBotsRemove} ${login}`}
          >
            {PR_FEEDBACK_SETTINGS_COPY.wizardBotsRemove}
          </Button>
        </li>
      ))}
    </ul>
  );
}

function ReviewBotCatalogPicker({
  selected,
  catalog,
  loading,
  loadError,
  onToggle,
  onAdd,
}: {
  selected: string[];
  catalog: ReviewBotOption[];
  loading?: boolean;
  loadError?: boolean;
  onToggle: (login: string) => void;
  onAdd: (login: string) => boolean;
}) {
  const [draft, setDraft] = useState("");
  const [manualAddOpen, setManualAddOpen] = useState(false);
  const selectedKeys = useMemo(() => new Set(selected.map((login) => login.toLowerCase())), [selected]);
  const rows = useMemo(() => reviewBotRows(catalog, selected), [catalog, selected]);
  const showFullLoading = Boolean(loading) && rows.length === 0;
  const showFoundMessage = !showFullLoading && !loadError && catalog.length > 0;

  const submitDraft = () => {
    if (!onAdd(draft)) {
      return;
    }
    setDraft("");
  };

  return (
    <section className="space-y-3">
      {loadError ? (
        <p className="workspace-body-text text-muted-foreground">{PR_FEEDBACK_SETTINGS_COPY.wizardBotsLoadError}</p>
      ) : null}
      {showFoundMessage ? (
        <p className="workspace-body-text text-muted-foreground" data-testid="discussion-setup-bots-found">
          {PR_FEEDBACK_SETTINGS_COPY.wizardBotsFound}
        </p>
      ) : null}
      <ReviewBotList rows={rows} selectedKeys={selectedKeys} showFullLoading={showFullLoading} onToggle={onToggle} />
      {manualAddOpen ? (
        <ReviewBotManualAdd
          draft={draft}
          label={PR_FEEDBACK_SETTINGS_COPY.wizardBotsAddLabel}
          placeholder={PR_FEEDBACK_SETTINGS_COPY.wizardBotsAddPlaceholder}
          onDraftChange={setDraft}
          onSubmit={submitDraft}
        />
      ) : (
        <ReviewBotManualPrompt onOpen={() => setManualAddOpen(true)} />
      )}
    </section>
  );
}

function ReviewBotList({
  rows,
  selectedKeys,
  showFullLoading,
  onToggle,
}: {
  rows: ReviewBotOption[];
  selectedKeys: Set<string>;
  showFullLoading: boolean;
  onToggle: (login: string) => void;
}) {
  return (
    <div
      className="max-h-56 overflow-y-auto rounded-lg border border-border"
      role="listbox"
      aria-label={PR_FEEDBACK_SETTINGS_COPY.wizardBotsLabel}
      aria-multiselectable="true"
      data-testid="discussion-setup-bots-picker"
    >
      {showFullLoading ? (
        <div
          className="flex items-center justify-center gap-2 px-4 py-6 text-muted-foreground"
          data-testid="discussion-setup-bots-loading"
        >
          <Loader2 className="size-4 animate-spin" aria-hidden />
          <span className="text-[13px]">{PR_FEEDBACK_SETTINGS_COPY.wizardBotsLoading}</span>
        </div>
      ) : rows.length === 0 ? (
        <p
          className="px-3 py-6 text-center text-[13px] text-muted-foreground"
          data-testid="discussion-setup-bots-empty"
        >
          {PR_FEEDBACK_SETTINGS_COPY.wizardBotsEmpty}
        </p>
      ) : (
        <ul className="divide-y divide-border" data-testid="discussion-setup-bots-list">
          {rows.map((bot) => {
            const isSelected = selectedKeys.has(bot.login.toLowerCase());
            return (
              <li key={bot.login}>
                <button
                  type="button"
                  role="option"
                  aria-selected={isSelected}
                  onClick={() => onToggle(bot.login)}
                  className={cn(
                    "flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors",
                    isSelected ? "bg-accent/50" : "hover:bg-accent/30",
                  )}
                  data-testid={`discussion-setup-bot-${bot.login}`}
                >
                  <span className="min-w-0 flex-1 truncate text-[13px] font-medium">{bot.displayName}</span>
                  <span className="flex size-3.5 shrink-0 items-center justify-center">
                    {isSelected ? <Check className="size-3.5 text-foreground" strokeWidth={2.5} aria-hidden /> : null}
                  </span>
                </button>
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

function ReviewBotManualAdd({
  draft,
  label,
  placeholder,
  onDraftChange,
  onSubmit,
}: {
  draft: string;
  label: string;
  placeholder: string;
  onDraftChange: (value: string) => void;
  onSubmit: () => void;
}) {
  return (
    <div className="review-bot-manual-add flex items-center gap-2">
      <Input
        id="discussion-setup-bot-manual"
        value={draft}
        placeholder={placeholder}
        aria-label={label}
        onChange={(event) => onDraftChange(event.target.value)}
        onKeyDown={(event) => {
          if (event.key === "Enter") {
            event.preventDefault();
            onSubmit();
          }
        }}
        data-testid="discussion-setup-bot-manual"
      />
      <Button
        type="button"
        variant="outline"
        disabled={normalizeReviewBotLogin(draft).length === 0}
        onClick={onSubmit}
        data-testid="discussion-setup-bot-add"
      >
        {PR_FEEDBACK_SETTINGS_COPY.wizardBotsAdd}
      </Button>
    </div>
  );
}

function ReviewBotManualPrompt({ onOpen }: { onOpen: () => void }) {
  return (
    <div className="space-y-2">
      <p className="workspace-body-text text-muted-foreground" data-testid="discussion-setup-bots-missing">
        {PR_FEEDBACK_SETTINGS_COPY.wizardBotsMissing}
      </p>
      <Button type="button" variant="outline" onClick={onOpen} data-testid="discussion-setup-bot-add-manually">
        {PR_FEEDBACK_SETTINGS_COPY.wizardBotsAddManually}
      </Button>
    </div>
  );
}

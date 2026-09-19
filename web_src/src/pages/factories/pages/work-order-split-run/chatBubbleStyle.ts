/**
 * Refine chat bubble styles. Every outgoing turn (the opening request, typed
 * notes, and survey picks) shares one primary bubble so the thread reads as
 * one person talking. `sp-chat-outgoing` (App.css) remaps the text tokens
 * inside the bubble so markdown and chips invert with it.
 */

/** Outgoing turn: primary fill, no frame, tighter bottom-right corner. */
export const USER_BUBBLE_CLASSNAME = "sp-chat-outgoing rounded-2xl rounded-br-md px-3.5 py-2.5 text-[14px] leading-5";

/** Collapsed-description fade that matches the primary bubble background. */
export const USER_BUBBLE_FADE_CLASSNAME = "from-primary via-primary/90";

/** The opening request uses the same bubble; it is the first outgoing turn. */
export const REQUEST_CARD_CLASSNAME = USER_BUBBLE_CLASSNAME;
export const REQUEST_CARD_FADE_CLASSNAME = USER_BUBBLE_FADE_CLASSNAME;

/** A survey pick: the same bubble at the shorter option size. */
export const SURVEY_PICK_CLASSNAME =
  "sp-chat-outgoing inline-block max-w-full rounded-2xl rounded-br-md px-3.5 py-2 text-left text-[13px] leading-5";

/** A skipped survey question: a dashed outline, so it is clearly not an answer. */
export const SURVEY_SKIPPED_CLASSNAME =
  "inline-block rounded-2xl rounded-br-md border border-dashed px-3.5 py-2 text-[13px] leading-5 text-muted-foreground";

/** Right-aligned column that holds one outgoing turn and its sender row. */
export const USER_TURN_CLASSNAME = "flex max-w-[85%] flex-col items-end gap-1";

/** Sender row above an outgoing turn. */
export const SENDER_ROW_CLASSNAME =
  "flex min-w-0 items-center gap-1.5 px-1 text-[11px] leading-4 text-muted-foreground";

import { CopyLinkButton } from "../../CopyLinkButton";
import { DuplicateWorkOrderButton } from "../../DuplicateWorkOrderButton";

const POPUP_HEADER_ACTION_BUTTON =
  "flex h-6 w-6 items-center justify-center rounded-full hover:bg-slate-950/5 dark:hover:bg-white/10";

export function WorkOrderPopupHeaderActions({
  copyUrl,
  canDuplicate,
  onDuplicate,
  duplicateBusy,
}: {
  copyUrl?: string;
  canDuplicate: boolean;
  onDuplicate: () => void;
  duplicateBusy: boolean;
}) {
  return (
    <>
      <CopyLinkButton
        url={copyUrl}
        className={POPUP_HEADER_ACTION_BUTTON}
        iconClassName="h-4 w-4"
        testId="popup-work-order-copy-link-button"
      />
      {canDuplicate ? (
        <DuplicateWorkOrderButton
          onClick={onDuplicate}
          busy={duplicateBusy}
          className={POPUP_HEADER_ACTION_BUTTON}
          iconClassName="h-4 w-4"
        />
      ) : null}
    </>
  );
}

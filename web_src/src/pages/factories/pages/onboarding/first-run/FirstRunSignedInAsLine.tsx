import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/ui/dropdownMenu";

import { FIRST_RUN_COPY } from "./firstRunCopy";

/** A linked provider account. The key is the id the provider uses to select it. */
export type FirstRunLinkedIdentity = { key: string; login: string };

export type FirstRunSignedInAsTestIds = { signedInAs: string; switchAccount: string };

/**
 * The connected provider account, with a menu to switch to another linked
 * account or to connect one more. GitHub and Bitbucket share this control.
 */
export function FirstRunSignedInAsLine({
  login,
  activeKey,
  identities,
  disabled,
  signedInText,
  connectAnotherLabel,
  testIds,
  onSelectIdentity,
  onConnectAnotherAccount,
}: {
  login: string;
  activeKey: string;
  identities: FirstRunLinkedIdentity[];
  disabled: boolean;
  signedInText: string;
  connectAnotherLabel: string;
  testIds: FirstRunSignedInAsTestIds;
  onSelectIdentity?: (key: string) => void;
  onConnectAnotherAccount?: () => void;
}) {
  const [before, after] = signedInText.split(login);
  return (
    <div className="flex items-center gap-2 text-[13px] leading-5">
      <p className="text-muted-foreground" data-testid={testIds.signedInAs}>
        {before}
        <span className="font-medium text-foreground">{login}</span>
        {after}
      </p>
      {onSelectIdentity && onConnectAnotherAccount ? (
        <AccountMenu
          activeKey={activeKey}
          identities={identities}
          disabled={disabled}
          connectAnotherLabel={connectAnotherLabel}
          testId={testIds.switchAccount}
          onSelectIdentity={onSelectIdentity}
          onConnectAnotherAccount={onConnectAnotherAccount}
        />
      ) : null}
    </div>
  );
}

function AccountMenu({
  activeKey,
  identities,
  disabled,
  connectAnotherLabel,
  testId,
  onSelectIdentity,
  onConnectAnotherAccount,
}: {
  activeKey: string;
  identities: FirstRunLinkedIdentity[];
  disabled: boolean;
  connectAnotherLabel: string;
  testId: string;
  onSelectIdentity: (key: string) => void;
  onConnectAnotherAccount: () => void;
}) {
  const linkedIdentities = identities.filter((identity) => Boolean(identity.key && identity.login));

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          type="button"
          variant="link"
          size="sm"
          className="h-auto p-0 text-[13px] font-normal text-muted-foreground hover:text-foreground"
          disabled={disabled}
          data-testid={testId}
        >
          {FIRST_RUN_COPY.choose.switchAccount}
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="start" className="w-60">
        <DropdownMenuRadioGroup
          value={activeKey}
          onValueChange={(nextKey) => {
            if (nextKey !== activeKey) onSelectIdentity(nextKey);
          }}
        >
          {linkedIdentities.map((identity) => (
            <DropdownMenuRadioItem key={identity.key} value={identity.key} className="text-[13px]">
              {identity.login}
            </DropdownMenuRadioItem>
          ))}
        </DropdownMenuRadioGroup>
        {linkedIdentities.length > 0 ? <DropdownMenuSeparator /> : null}
        <DropdownMenuItem className="text-[13px]" onSelect={onConnectAnotherAccount}>
          {connectAnotherLabel}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

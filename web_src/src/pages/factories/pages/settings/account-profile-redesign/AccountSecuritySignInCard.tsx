import { Button } from "@/components/ui/button";

import { FactorySettingsCard } from "../FactorySettingsCard";
import { SettingsActionRow } from "./accountProfileRedesignParts";

export function AccountSecuritySignInCard({
  passwordSet,
  onChangePassword,
}: {
  passwordSet: boolean;
  onChangePassword: () => void;
}) {
  if (!passwordSet) {
    return null;
  }

  return (
    <FactorySettingsCard title="Sign in methods" data-testid="account-redesign-signin">
      <p className="text-[12px] text-muted-foreground">Change the password for this SuperPlane account.</p>
      <ul className="mt-4 space-y-4">
        <li>
          <SettingsActionRow
            title="Password"
            description="Password is set."
            testId="account-redesign-password"
            action={
              <Button type="button" size="sm" variant="outline" onClick={onChangePassword}>
                Change password
              </Button>
            }
          />
        </li>
      </ul>
    </FactorySettingsCard>
  );
}

import { usePageTitle } from "@/hooks/usePageTitle";
import { HomePageShell } from "./HomePageShell";
import { ZeroStatePage } from "./ZeroStatePage";

export function NewAppPage() {
  usePageTitle(["New App"]);

  return (
    <HomePageShell>
      <ZeroStatePage />
    </HomePageShell>
  );
}

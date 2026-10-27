export function OrganizationSuspensionBanner() {
  return (
    <div
      role="status"
      className="shrink-0 bg-amber-100 px-4 py-2 text-center text-sm text-amber-950 dark:bg-amber-950 dark:text-amber-100"
    >
      This organization is suspended. You can view existing data, but cannot start new work. Contact support for help.
    </div>
  );
}

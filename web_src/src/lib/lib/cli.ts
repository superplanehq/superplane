export const CLI_INSTALL_COMMAND = "curl -fsSL https://install.superplane.com/install.sh | sh";

export function getInstallCommand(): string {
  return CLI_INSTALL_COMMAND;
}

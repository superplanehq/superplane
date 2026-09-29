const PUBLIC_FACTORY_LINE_PATH = /^\/[^/]+\/workspaces\/[^/]+\/lines\/[^/]+$/;

/** True for one line board URL. Edit, setup, and task URLs stay private. */
export function isPublicFactoryLinePath(pathname: string): boolean {
  return PUBLIC_FACTORY_LINE_PATH.test(pathname);
}

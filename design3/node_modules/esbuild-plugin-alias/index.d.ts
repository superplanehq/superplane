declare module 'esbuild-plugin-alias' {
  import type { Plugin } from 'esbuild';
  import alias from 'esbuild-plugin-alias';

  const aliasTyped: (opts: Record<string, string>) => Plugin = alias;

  export default aliasTyped;
}

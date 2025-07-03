import { TransformOptions, ParserOptions } from '@babel/core';
import { ResolvedConfig, PluginOption } from 'vite';

interface Options {
    include?: string | RegExp | Array<string | RegExp>;
    exclude?: string | RegExp | Array<string | RegExp>;
    /**
     * Enable `react-refresh` integration. Vite disables this in prod env or build mode.
     * @default true
     */
    fastRefresh?: boolean;
    /**
     * Set this to `"automatic"` to use [vite-react-jsx](https://github.com/alloc/vite-react-jsx).
     * @default "automatic"
     */
    jsxRuntime?: 'classic' | 'automatic';
    /**
     * Control where the JSX factory is imported from.
     * This option is ignored when `jsxRuntime` is not `"automatic"`.
     * @default "react"
     */
    jsxImportSource?: string;
    /**
     * Set this to `true` to annotate the JSX factory with `\/* @__PURE__ *\/`.
     * This option is ignored when `jsxRuntime` is not `"automatic"`.
     * @default true
     */
    jsxPure?: boolean;
    /**
     * Babel configuration applied in both dev and prod.
     */
    babel?: BabelOptions | ((id: string, options: {
        ssr?: boolean;
    }) => BabelOptions);
}
type BabelOptions = Omit<TransformOptions, 'ast' | 'filename' | 'root' | 'sourceFileName' | 'sourceMaps' | 'inputSourceMap'>;
/**
 * The object type used by the `options` passed to plugins with
 * an `api.reactBabel` method.
 */
interface ReactBabelOptions extends BabelOptions {
    plugins: Extract<BabelOptions['plugins'], any[]>;
    presets: Extract<BabelOptions['presets'], any[]>;
    overrides: Extract<BabelOptions['overrides'], any[]>;
    parserOpts: ParserOptions & {
        plugins: Extract<ParserOptions['plugins'], any[]>;
    };
}
type ReactBabelHook = (babelConfig: ReactBabelOptions, context: ReactBabelHookContext, config: ResolvedConfig) => void;
type ReactBabelHookContext = {
    ssr: boolean;
    id: string;
};
declare module 'vite' {
    interface Plugin {
        api?: {
            /**
             * Manipulate the Babel options of `@vitejs/plugin-react`
             */
            reactBabel?: ReactBabelHook;
        };
    }
}
declare function viteReact(opts?: Options): PluginOption[];
declare namespace viteReact {
    var preambleCode: string;
}

export { BabelOptions, Options, ReactBabelOptions, viteReact as default };

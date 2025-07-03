declare type ModuleType = "esm" | "cjs";

/**
 * Information that discribes a module to be imported.
 */
declare type ModuleInfo = {
  /**
   * Global variable name with which the import statements of the module
   * should be replaced.
   */
  varName: string;

  /**
   * Type (either `"esm"` or `"cjs"`) that determines the internal behavior of
   * this plugin. Defaults to `"esm"`.
   */
  type?: ModuleType;

  /**
   * Names of variables that are exported from the module and may be imported
   * from another module.
   * No effect if `type` is `"cjs"`.
   */
  namedExports?: readonly string[];

  /**
   * Set to `false` to prevent emitting code for default import/export
   * (which you won't need to unless you are finicky).
   * Defaults to `true`. No effect if `type` is `"cjs"`.
   */
  defaultExport?: boolean;
};

declare const globalsModuleInfoMap: Required<Record<"react" | "react-dom" | "@storybook/components" | "@storybook/channels" | "@storybook/core-events" | "@storybook/router" | "@storybook/theming" | "@storybook/api" | "@storybook/manager-api" | "@storybook/addons" | "@storybook/client-logger" | "@storybook/types", Required<ModuleInfo>>>;

export { globalsModuleInfoMap };

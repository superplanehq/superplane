import { S as StorybookError } from '../storybook-error-32ac504c.js';

/**
 * If you can't find a suitable category for your error, create one
 * based on the package name/file path of which the error is thrown.
 * For instance:
 * If it's from @storybook/node-logger, then NODE-LOGGER
 * If it's from a package that is too broad, e.g. @storybook/cli in the init command, then use a combination like CLI_INIT
 */
declare enum Category {
    CLI = "CLI",
    CLI_INIT = "CLI_INIT",
    CLI_AUTOMIGRATE = "CLI_AUTOMIGRATE",
    CLI_UPGRADE = "CLI_UPGRADE",
    CLI_ADD = "CLI_ADD",
    CODEMOD = "CODEMOD",
    CORE_SERVER = "CORE-SERVER",
    CSF_PLUGIN = "CSF-PLUGIN",
    CSF_TOOLS = "CSF-TOOLS",
    CORE_COMMON = "CORE-COMMON",
    NODE_LOGGER = "NODE-LOGGER",
    TELEMETRY = "TELEMETRY",
    BUILDER_MANAGER = "BUILDER-MANAGER",
    BUILDER_VITE = "BUILDER-VITE",
    BUILDER_WEBPACK5 = "BUILDER-WEBPACK5",
    SOURCE_LOADER = "SOURCE-LOADER",
    POSTINSTALL = "POSTINSTALL",
    DOCS_TOOLS = "DOCS-TOOLS",
    CORE_WEBPACK = "CORE-WEBPACK",
    FRAMEWORK_ANGULAR = "FRAMEWORK_ANGULAR",
    FRAMEWORK_EMBER = "FRAMEWORK_EMBER",
    FRAMEWORK_HTML_VITE = "FRAMEWORK_HTML-VITE",
    FRAMEWORK_HTML_WEBPACK5 = "FRAMEWORK_HTML-WEBPACK5",
    FRAMEWORK_NEXTJS = "FRAMEWORK_NEXTJS",
    FRAMEWORK_PREACT_VITE = "FRAMEWORK_PREACT-VITE",
    FRAMEWORK_PREACT_WEBPACK5 = "FRAMEWORK_PREACT-WEBPACK5",
    FRAMEWORK_REACT_VITE = "FRAMEWORK_REACT-VITE",
    FRAMEWORK_REACT_WEBPACK5 = "FRAMEWORK_REACT-WEBPACK5",
    FRAMEWORK_SERVER_WEBPACK5 = "FRAMEWORK_SERVER-WEBPACK5",
    FRAMEWORK_SVELTE_VITE = "FRAMEWORK_SVELTE-VITE",
    FRAMEWORK_SVELTE_WEBPACK5 = "FRAMEWORK_SVELTE-WEBPACK5",
    FRAMEWORK_SVELTEKIT = "FRAMEWORK_SVELTEKIT",
    FRAMEWORK_VUE_VITE = "FRAMEWORK_VUE-VITE",
    FRAMEWORK_VUE_WEBPACK5 = "FRAMEWORK_VUE-WEBPACK5",
    FRAMEWORK_VUE3_VITE = "FRAMEWORK_VUE3-VITE",
    FRAMEWORK_VUE3_WEBPACK5 = "FRAMEWORK_VUE3-WEBPACK5",
    FRAMEWORK_WEB_COMPONENTS_VITE = "FRAMEWORK_WEB-COMPONENTS-VITE",
    FRAMEWORK_WEB_COMPONENTS_WEBPACK5 = "FRAMEWORK_WEB-COMPONENTS-WEBPACK5"
}
declare class NxProjectDetectedError extends StorybookError {
    readonly category = Category.CLI_INIT;
    readonly code = 1;
    readonly documentation = "https://nx.dev/packages/storybook";
    template(): string;
}
declare class MissingFrameworkFieldError extends StorybookError {
    readonly category = Category.CORE_COMMON;
    readonly code = 1;
    readonly documentation = "https://github.com/storybookjs/storybook/blob/next/MIGRATION.md#new-framework-api";
    template(): string;
}
declare class InvalidFrameworkNameError extends StorybookError {
    data: {
        frameworkName: string;
    };
    readonly category = Category.CORE_COMMON;
    readonly code = 2;
    readonly documentation = "https://github.com/storybookjs/storybook/blob/next/MIGRATION.md#new-framework-api";
    constructor(data: {
        frameworkName: string;
    });
    template(): string;
}
declare class CouldNotEvaluateFrameworkError extends StorybookError {
    data: {
        frameworkName: string;
    };
    readonly category = Category.CORE_COMMON;
    readonly code = 3;
    constructor(data: {
        frameworkName: string;
    });
    template(): string;
}
declare class ConflictingStaticDirConfigError extends StorybookError {
    readonly category = Category.CORE_SERVER;
    readonly code = 1;
    readonly documentation = "https://storybook.js.org/docs/react/configure/images-and-assets#serving-static-files-via-storybook-configuration";
    template(): string;
}
declare class InvalidStoriesEntryError extends StorybookError {
    readonly category = Category.CORE_COMMON;
    readonly code = 4;
    readonly documentation = "https://storybook.js.org/docs/react/faq#can-i-have-a-storybook-with-no-local-stories";
    template(): string;
}
declare class WebpackMissingStatsError extends StorybookError {
    readonly category = Category.BUILDER_WEBPACK5;
    readonly code = 1;
    documentation: string[];
    template(): string;
}
declare class WebpackInvocationError extends StorybookError {
    data: {
        error: Error;
    };
    readonly category = Category.BUILDER_WEBPACK5;
    readonly code = 2;
    private errorMessage;
    constructor(data: {
        error: Error;
    });
    template(): string;
}
declare class WebpackCompilationError extends StorybookError {
    data: {
        errors: {
            message: string;
            stack?: string;
            name?: string;
        }[];
    };
    readonly category = Category.BUILDER_WEBPACK5;
    readonly code = 3;
    constructor(data: {
        errors: {
            message: string;
            stack?: string;
            name?: string;
        }[];
    });
    template(): string;
}
declare class MissingAngularJsonError extends StorybookError {
    data: {
        path: string;
    };
    readonly category = Category.CLI_INIT;
    readonly code = 2;
    readonly documentation = "https://storybook.js.org/docs/angular/faq#error-no-angularjson-file-found";
    constructor(data: {
        path: string;
    });
    template(): string;
}
declare class AngularLegacyBuildOptionsError extends StorybookError {
    readonly category = Category.FRAMEWORK_ANGULAR;
    readonly code = 1;
    readonly documentation: string[];
    template(): string;
}
declare class CriticalPresetLoadError extends StorybookError {
    data: {
        error: Error;
        presetName: string;
    };
    readonly category = Category.CORE_SERVER;
    readonly code = 2;
    constructor(data: {
        error: Error;
        presetName: string;
    });
    template(): string;
}
declare class MissingBuilderError extends StorybookError {
    readonly category = Category.CORE_SERVER;
    readonly code = 3;
    readonly documentation = "https://github.com/storybookjs/storybook/issues/24071";
    template(): string;
}
declare class GoogleFontsDownloadError extends StorybookError {
    data: {
        fontFamily: string;
        url: string;
    };
    readonly category = Category.FRAMEWORK_NEXTJS;
    readonly code = 1;
    readonly documentation = "https://github.com/storybookjs/storybook/blob/next/code/frameworks/nextjs/README.md#nextjs-font-optimization";
    constructor(data: {
        fontFamily: string;
        url: string;
    });
    template(): string;
}
declare class GoogleFontsLoadingError extends StorybookError {
    data: {
        error: unknown | Error;
        url: string;
    };
    readonly category = Category.FRAMEWORK_NEXTJS;
    readonly code = 2;
    readonly documentation = "https://github.com/storybookjs/storybook/blob/next/code/frameworks/nextjs/README.md#nextjs-font-optimization";
    constructor(data: {
        error: unknown | Error;
        url: string;
    });
    template(): string;
}
declare class NextjsSWCNotSupportedError extends StorybookError {
    readonly category = Category.FRAMEWORK_NEXTJS;
    readonly code = 3;
    readonly documentation = "https://github.com/storybookjs/storybook/blob/next/code/frameworks/nextjs/README.md#manual-migration";
    template(): string;
}
declare class NoMatchingExportError extends StorybookError {
    data: {
        error: unknown | Error;
    };
    readonly category = Category.CORE_SERVER;
    readonly code = 4;
    constructor(data: {
        error: unknown | Error;
    });
    template(): string;
}
declare class UpgradeStorybookToLowerVersionError extends StorybookError {
    data: {
        beforeVersion: string;
        currentVersion: string;
    };
    readonly category = Category.CLI_UPGRADE;
    readonly code = 3;
    constructor(data: {
        beforeVersion: string;
        currentVersion: string;
    });
    template(): string;
}
declare class UpgradeStorybookToSameVersionError extends StorybookError {
    data: {
        beforeVersion: string;
    };
    readonly category = Category.CLI_UPGRADE;
    readonly code = 4;
    constructor(data: {
        beforeVersion: string;
    });
    template(): string;
}

export { AngularLegacyBuildOptionsError, Category, ConflictingStaticDirConfigError, CouldNotEvaluateFrameworkError, CriticalPresetLoadError, GoogleFontsDownloadError, GoogleFontsLoadingError, InvalidFrameworkNameError, InvalidStoriesEntryError, MissingAngularJsonError, MissingBuilderError, MissingFrameworkFieldError, NextjsSWCNotSupportedError, NoMatchingExportError, NxProjectDetectedError, UpgradeStorybookToLowerVersionError, UpgradeStorybookToSameVersionError, WebpackCompilationError, WebpackInvocationError, WebpackMissingStatsError };

declare const globalsNameReferenceMap: {
    readonly react: "__REACT__";
    readonly 'react-dom': "__REACT_DOM__";
    readonly '@storybook/components': "__STORYBOOK_COMPONENTS__";
    readonly '@storybook/channels': "__STORYBOOK_CHANNELS__";
    readonly '@storybook/core-events': "__STORYBOOK_CORE_EVENTS__";
    readonly '@storybook/router': "__STORYBOOK_ROUTER__";
    readonly '@storybook/theming': "__STORYBOOK_THEMING__";
    readonly '@storybook/api': "__STORYBOOK_API__";
    readonly '@storybook/manager-api': "__STORYBOOK_API__";
    readonly '@storybook/addons': "__STORYBOOK_ADDONS__";
    readonly '@storybook/client-logger': "__STORYBOOK_CLIENT_LOGGER__";
    readonly '@storybook/types': "__STORYBOOK_TYPES__";
};
declare const globalPackages: ("react" | "react-dom" | "@storybook/components" | "@storybook/channels" | "@storybook/core-events" | "@storybook/router" | "@storybook/theming" | "@storybook/api" | "@storybook/manager-api" | "@storybook/addons" | "@storybook/client-logger" | "@storybook/types")[];

export { globalPackages, globalsNameReferenceMap };

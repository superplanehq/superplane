"use strict";var __create=Object.create;var __defProp=Object.defineProperty;var __getOwnPropDesc=Object.getOwnPropertyDescriptor;var __getOwnPropNames=Object.getOwnPropertyNames;var __getProtoOf=Object.getPrototypeOf,__hasOwnProp=Object.prototype.hasOwnProperty;var __export=(target,all)=>{for(var name in all)__defProp(target,name,{get:all[name],enumerable:!0})},__copyProps=(to,from,except,desc)=>{if(from&&typeof from=="object"||typeof from=="function")for(let key of __getOwnPropNames(from))!__hasOwnProp.call(to,key)&&key!==except&&__defProp(to,key,{get:()=>from[key],enumerable:!(desc=__getOwnPropDesc(from,key))||desc.enumerable});return to};var __toESM=(mod,isNodeMode,target)=>(target=mod!=null?__create(__getProtoOf(mod)):{},__copyProps(isNodeMode||!mod||!mod.__esModule?__defProp(target,"default",{value:mod,enumerable:!0}):target,mod)),__toCommonJS=mod=>__copyProps(__defProp({},"__esModule",{value:!0}),mod);var server_errors_exports={};__export(server_errors_exports,{AngularLegacyBuildOptionsError:()=>AngularLegacyBuildOptionsError,Category:()=>Category,ConflictingStaticDirConfigError:()=>ConflictingStaticDirConfigError,CouldNotEvaluateFrameworkError:()=>CouldNotEvaluateFrameworkError,CriticalPresetLoadError:()=>CriticalPresetLoadError,GoogleFontsDownloadError:()=>GoogleFontsDownloadError,GoogleFontsLoadingError:()=>GoogleFontsLoadingError,InvalidFrameworkNameError:()=>InvalidFrameworkNameError,InvalidStoriesEntryError:()=>InvalidStoriesEntryError,MissingAngularJsonError:()=>MissingAngularJsonError,MissingBuilderError:()=>MissingBuilderError,MissingFrameworkFieldError:()=>MissingFrameworkFieldError,NextjsSWCNotSupportedError:()=>NextjsSWCNotSupportedError,NoMatchingExportError:()=>NoMatchingExportError,NxProjectDetectedError:()=>NxProjectDetectedError,UpgradeStorybookToLowerVersionError:()=>UpgradeStorybookToLowerVersionError,UpgradeStorybookToSameVersionError:()=>UpgradeStorybookToSameVersionError,WebpackCompilationError:()=>WebpackCompilationError,WebpackInvocationError:()=>WebpackInvocationError,WebpackMissingStatsError:()=>WebpackMissingStatsError});module.exports=__toCommonJS(server_errors_exports);var import_ts_dedent=__toESM(require("ts-dedent"));var StorybookError=class extends Error{constructor(){super(...arguments);this.data={};this.documentation=!1;this.fromStorybook=!0}get fullErrorCode(){let paddedCode=String(this.code).padStart(4,"0");return`SB_${this.category}_${paddedCode}`}get name(){let errorName=this.constructor.name;return`${this.fullErrorCode} (${errorName})`}get message(){let page;return this.documentation===!0?page=`https://storybook.js.org/error/${this.fullErrorCode}`:typeof this.documentation=="string"?page=this.documentation:Array.isArray(this.documentation)&&(page=`
${this.documentation.map(doc=>`	- ${doc}`).join(`
`)}`),`${this.template()}${page!=null?`

More info: ${page}
`:""}`}};var Category=(Category2=>(Category2.CLI="CLI",Category2.CLI_INIT="CLI_INIT",Category2.CLI_AUTOMIGRATE="CLI_AUTOMIGRATE",Category2.CLI_UPGRADE="CLI_UPGRADE",Category2.CLI_ADD="CLI_ADD",Category2.CODEMOD="CODEMOD",Category2.CORE_SERVER="CORE-SERVER",Category2.CSF_PLUGIN="CSF-PLUGIN",Category2.CSF_TOOLS="CSF-TOOLS",Category2.CORE_COMMON="CORE-COMMON",Category2.NODE_LOGGER="NODE-LOGGER",Category2.TELEMETRY="TELEMETRY",Category2.BUILDER_MANAGER="BUILDER-MANAGER",Category2.BUILDER_VITE="BUILDER-VITE",Category2.BUILDER_WEBPACK5="BUILDER-WEBPACK5",Category2.SOURCE_LOADER="SOURCE-LOADER",Category2.POSTINSTALL="POSTINSTALL",Category2.DOCS_TOOLS="DOCS-TOOLS",Category2.CORE_WEBPACK="CORE-WEBPACK",Category2.FRAMEWORK_ANGULAR="FRAMEWORK_ANGULAR",Category2.FRAMEWORK_EMBER="FRAMEWORK_EMBER",Category2.FRAMEWORK_HTML_VITE="FRAMEWORK_HTML-VITE",Category2.FRAMEWORK_HTML_WEBPACK5="FRAMEWORK_HTML-WEBPACK5",Category2.FRAMEWORK_NEXTJS="FRAMEWORK_NEXTJS",Category2.FRAMEWORK_PREACT_VITE="FRAMEWORK_PREACT-VITE",Category2.FRAMEWORK_PREACT_WEBPACK5="FRAMEWORK_PREACT-WEBPACK5",Category2.FRAMEWORK_REACT_VITE="FRAMEWORK_REACT-VITE",Category2.FRAMEWORK_REACT_WEBPACK5="FRAMEWORK_REACT-WEBPACK5",Category2.FRAMEWORK_SERVER_WEBPACK5="FRAMEWORK_SERVER-WEBPACK5",Category2.FRAMEWORK_SVELTE_VITE="FRAMEWORK_SVELTE-VITE",Category2.FRAMEWORK_SVELTE_WEBPACK5="FRAMEWORK_SVELTE-WEBPACK5",Category2.FRAMEWORK_SVELTEKIT="FRAMEWORK_SVELTEKIT",Category2.FRAMEWORK_VUE_VITE="FRAMEWORK_VUE-VITE",Category2.FRAMEWORK_VUE_WEBPACK5="FRAMEWORK_VUE-WEBPACK5",Category2.FRAMEWORK_VUE3_VITE="FRAMEWORK_VUE3-VITE",Category2.FRAMEWORK_VUE3_WEBPACK5="FRAMEWORK_VUE3-WEBPACK5",Category2.FRAMEWORK_WEB_COMPONENTS_VITE="FRAMEWORK_WEB-COMPONENTS-VITE",Category2.FRAMEWORK_WEB_COMPONENTS_WEBPACK5="FRAMEWORK_WEB-COMPONENTS-WEBPACK5",Category2))(Category||{}),NxProjectDetectedError=class extends StorybookError{constructor(){super(...arguments);this.category="CLI_INIT";this.code=1;this.documentation="https://nx.dev/packages/storybook"}template(){return import_ts_dedent.default`
      We have detected Nx in your project. Nx has its own Storybook initializer, so please use it instead.
      Run "nx g @nx/storybook:configuration" to add Storybook to your project.
    `}},MissingFrameworkFieldError=class extends StorybookError{constructor(){super(...arguments);this.category="CORE-COMMON";this.code=1;this.documentation="https://github.com/storybookjs/storybook/blob/next/MIGRATION.md#new-framework-api"}template(){return import_ts_dedent.default`
      Could not find a 'framework' field in Storybook config.

      Please run 'npx storybook@next automigrate' to automatically fix your config.
    `}},InvalidFrameworkNameError=class extends StorybookError{constructor(data){super();this.data=data;this.category="CORE-COMMON";this.code=2;this.documentation="https://github.com/storybookjs/storybook/blob/next/MIGRATION.md#new-framework-api"}template(){return import_ts_dedent.default`
      Invalid value of '${this.data.frameworkName}' in the 'framework' field of Storybook config.

      Please run 'npx storybook@next automigrate' to automatically fix your config.
    `}},CouldNotEvaluateFrameworkError=class extends StorybookError{constructor(data){super();this.data=data;this.category="CORE-COMMON";this.code=3}template(){return import_ts_dedent.default`
      Could not evaluate the '${this.data.frameworkName}' package from the 'framework' field of Storybook config.

      Are you sure it's a valid package and is installed?
    `}},ConflictingStaticDirConfigError=class extends StorybookError{constructor(){super(...arguments);this.category="CORE-SERVER";this.code=1;this.documentation="https://storybook.js.org/docs/react/configure/images-and-assets#serving-static-files-via-storybook-configuration"}template(){return import_ts_dedent.default`
      Storybook encountered a conflict when trying to serve statics. You have configured both:
      * Storybook's option in the config file: 'staticDirs'
      * Storybook's (deprecated) CLI flag: '--staticDir' or '-s'
      
      Please remove the CLI flag from your storybook script and use only the 'staticDirs' option instead.
    `}},InvalidStoriesEntryError=class extends StorybookError{constructor(){super(...arguments);this.category="CORE-COMMON";this.code=4;this.documentation="https://storybook.js.org/docs/react/faq#can-i-have-a-storybook-with-no-local-stories"}template(){return import_ts_dedent.default`
      Storybook could not index your stories.
      Your main configuration somehow does not contain a 'stories' field, or it resolved to an empty array.

      Please check your main configuration file and make sure it exports a 'stories' field that is not an empty array.
    `}},WebpackMissingStatsError=class extends StorybookError{constructor(){super(...arguments);this.category="BUILDER-WEBPACK5";this.code=1;this.documentation=["https://webpack.js.org/configuration/stats/","https://storybook.js.org/docs/react/builders/webpack#configure"]}template(){return import_ts_dedent.default`
      No Webpack stats found. Did you turn off stats reporting in your webpack config?
      Storybook needs Webpack stats (including errors) in order to build correctly.
    `}},WebpackInvocationError=class extends StorybookError{constructor(data){super();this.data=data;this.category="BUILDER-WEBPACK5";this.code=2;this.errorMessage="";this.errorMessage=data.error.message}template(){return this.errorMessage.trim()}};function removeAnsiEscapeCodes(input=""){return input.replace(/\u001B\[[0-9;]*m/g,"")}var WebpackCompilationError=class extends StorybookError{constructor(data){super();this.data=data;this.category="BUILDER-WEBPACK5";this.code=3;this.data.errors=data.errors.map(err=>({...err,message:removeAnsiEscapeCodes(err.message),stack:removeAnsiEscapeCodes(err.stack),name:err.name}))}template(){return import_ts_dedent.default`
      There were problems when compiling your code with Webpack.
      Run Storybook with --debug-webpack for more information.
    `}},MissingAngularJsonError=class extends StorybookError{constructor(data){super();this.data=data;this.category="CLI_INIT";this.code=2;this.documentation="https://storybook.js.org/docs/angular/faq#error-no-angularjson-file-found"}template(){return import_ts_dedent.default`
      An angular.json file was not found in the current working directory: ${this.data.path}
      Storybook needs it to work properly, so please rerun the command at the root of your project, where the angular.json file is located.
    `}},AngularLegacyBuildOptionsError=class extends StorybookError{constructor(){super(...arguments);this.category="FRAMEWORK_ANGULAR";this.code=1;this.documentation=["https://github.com/storybookjs/storybook/blob/next/MIGRATION.md#angular-drop-support-for-calling-storybook-directly","https://github.com/storybookjs/storybook/tree/next/code/frameworks/angular#how-do-i-migrate-to-an-angular-storybook-builder"]}template(){return import_ts_dedent.default`
      Your Storybook startup script uses a solution that is not supported anymore.
      You must use Angular builder to have an explicit configuration on the project used in angular.json.

      Please run 'npx storybook@next automigrate' to automatically fix your config.
    `}},CriticalPresetLoadError=class extends StorybookError{constructor(data){super();this.data=data;this.category="CORE-SERVER";this.code=2}template(){return import_ts_dedent.default`
      Storybook failed to load the following preset: ${this.data.presetName}.

      Please check whether your setup is correct, the Storybook dependencies (and their peer dependencies) are installed correctly and there are no package version clashes.

      If you believe this is a bug, please open an issue on Github.

      ${this.data.error.stack||this.data.error.message}
    `}},MissingBuilderError=class extends StorybookError{constructor(){super(...arguments);this.category="CORE-SERVER";this.code=3;this.documentation="https://github.com/storybookjs/storybook/issues/24071"}template(){return import_ts_dedent.default`
      Storybook could not find a builder configuration for your project. 
      Builders normally come from a framework package e.g. '@storybook/react-vite', or from builder packages e.g. '@storybook/builder-vite'.
      
      - Does your main config file contain a 'framework' field configured correctly?
      - Is the Storybook framework package installed correctly?
      - If you don't use a framework, does your main config contain a 'core.builder' configured correctly?
      - Are you in a monorepo and perhaps the framework package is hoisted incorrectly?

      If you believe this is a bug, please describe your issue in detail on Github.
    `}},GoogleFontsDownloadError=class extends StorybookError{constructor(data){super();this.data=data;this.category="FRAMEWORK_NEXTJS";this.code=1;this.documentation="https://github.com/storybookjs/storybook/blob/next/code/frameworks/nextjs/README.md#nextjs-font-optimization"}template(){return import_ts_dedent.default`
      Failed to fetch \`${this.data.fontFamily}\` from Google Fonts with URL: \`${this.data.url}\`
    `}},GoogleFontsLoadingError=class extends StorybookError{constructor(data){super();this.data=data;this.category="FRAMEWORK_NEXTJS";this.code=2;this.documentation="https://github.com/storybookjs/storybook/blob/next/code/frameworks/nextjs/README.md#nextjs-font-optimization"}template(){return import_ts_dedent.default`
      An error occurred when trying to load Google Fonts with URL \`${this.data.url}\`.
      
      ${this.data.error instanceof Error?this.data.error.message:""}
    `}},NextjsSWCNotSupportedError=class extends StorybookError{constructor(){super(...arguments);this.category="FRAMEWORK_NEXTJS";this.code=3;this.documentation="https://github.com/storybookjs/storybook/blob/next/code/frameworks/nextjs/README.md#manual-migration"}template(){return import_ts_dedent.default`
    You have activated the SWC mode for Next.js, but you are not using Next.js 14.0.0 or higher. 
    SWC is only supported in Next.js 14.0.0 and higher. Please go to your .storybook/main.<js|ts> file
    and remove the { framework: { options: { builder: { useSWC: true } } } } option or upgrade to Next.js v14 or later.
    `}},NoMatchingExportError=class extends StorybookError{constructor(data){super();this.data=data;this.category="CORE-SERVER";this.code=4}template(){return import_ts_dedent.default`
      There was an exports mismatch error when trying to build Storybook.
      Please check whether the versions of your Storybook packages match whenever possible, as this might be the cause.
      
      Problematic example:
      { "@storybook/react": "7.5.3", "@storybook/react-vite": "7.4.5", "storybook": "7.3.0" }

      Correct example:
      { "@storybook/react": "7.5.3", "@storybook/react-vite": "7.5.3", "storybook": "7.5.3" }

      Please run \`npx storybook@latest doctor\` for guidance on how to fix this issue.
    `}},UpgradeStorybookToLowerVersionError=class extends StorybookError{constructor(data){super();this.data=data;this.category="CLI_UPGRADE";this.code=3}template(){return import_ts_dedent.default`
      You are trying to upgrade Storybook to a lower version than the version currently installed. This is not supported.
      Storybook version ${this.data.beforeVersion} was detected in your project, but you are trying to "upgrade" to version ${this.data.currentVersion}.
      
      This usually happens when running the upgrade command without a version specifier, e.g. "npx storybook upgrade".
      This will cause npm to run the globally cached storybook binary, which might be an older version.
      Instead you should always run the Storybook CLI with a version specifier to force npm to download the latest version:
      
      "npx storybook@latest upgrade"
    `}},UpgradeStorybookToSameVersionError=class extends StorybookError{constructor(data){super();this.data=data;this.category="CLI_UPGRADE";this.code=4}template(){return import_ts_dedent.default`
      You are trying to upgrade Storybook to the same version that is currently installed in the project, version ${this.data.beforeVersion}. This is not supported.
      
      This usually happens when running the upgrade command without a version specifier, e.g. "npx storybook upgrade".
      This will cause npm to run the globally cached storybook binary, which might be the same version that you already have.
      This also happens if you're running the Storybook CLI that is locally installed in your project.
      If you intended to upgrade to the latest version, you should always run the Storybook CLI with a version specifier to force npm to download the latest version:
      "npx storybook@latest upgrade"
      If you intended to re-run automigrations, you should run the "automigrate" command directly instead:
      "npx storybook@${this.data.beforeVersion} automigrate"
    `}};0&&(module.exports={AngularLegacyBuildOptionsError,Category,ConflictingStaticDirConfigError,CouldNotEvaluateFrameworkError,CriticalPresetLoadError,GoogleFontsDownloadError,GoogleFontsLoadingError,InvalidFrameworkNameError,InvalidStoriesEntryError,MissingAngularJsonError,MissingBuilderError,MissingFrameworkFieldError,NextjsSWCNotSupportedError,NoMatchingExportError,NxProjectDetectedError,UpgradeStorybookToLowerVersionError,UpgradeStorybookToSameVersionError,WebpackCompilationError,WebpackInvocationError,WebpackMissingStatsError});

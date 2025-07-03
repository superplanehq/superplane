import { StorybookError } from '../chunk-3FIG6PJN.mjs';
import dedent from 'ts-dedent';

var Category=(Category2=>(Category2.PREVIEW_CLIENT_LOGGER="PREVIEW_CLIENT-LOGGER",Category2.PREVIEW_CHANNELS="PREVIEW_CHANNELS",Category2.PREVIEW_CORE_EVENTS="PREVIEW_CORE-EVENTS",Category2.PREVIEW_INSTRUMENTER="PREVIEW_INSTRUMENTER",Category2.PREVIEW_API="PREVIEW_API",Category2.PREVIEW_REACT_DOM_SHIM="PREVIEW_REACT-DOM-SHIM",Category2.PREVIEW_ROUTER="PREVIEW_ROUTER",Category2.PREVIEW_THEMING="PREVIEW_THEMING",Category2.RENDERER_HTML="RENDERER_HTML",Category2.RENDERER_PREACT="RENDERER_PREACT",Category2.RENDERER_REACT="RENDERER_REACT",Category2.RENDERER_SERVER="RENDERER_SERVER",Category2.RENDERER_SVELTE="RENDERER_SVELTE",Category2.RENDERER_VUE="RENDERER_VUE",Category2.RENDERER_VUE3="RENDERER_VUE3",Category2.RENDERER_WEB_COMPONENTS="RENDERER_WEB-COMPONENTS",Category2))(Category||{}),MissingStoryAfterHmrError=class extends StorybookError{constructor(data){super();this.data=data;this.category="PREVIEW_API";this.code=1;}template(){return dedent`
    Couldn't find story matching id '${this.data.storyId}' after HMR.
    - Did you just rename a story?
    - Did you remove it from your CSF file?
    - Are you sure a story with the id '${this.data.storyId}' exists?
    - Please check the values in the stories field of your main.js config and see if they would match your CSF File.
    - Also check the browser console and terminal for potential error messages.`}},ImplicitActionsDuringRendering=class extends StorybookError{constructor(data){super();this.data=data;this.category="PREVIEW_API";this.code=2;this.documentation="https://github.com/storybookjs/storybook/blob/next/MIGRATION.md#using-implicit-actions-during-rendering-is-deprecated-for-example-in-the-play-function";}template(){return dedent`
      We detected that you use an implicit action arg during ${this.data.phase} of your story.  
      ${this.data.deprecated?`
This is deprecated and won't work in Storybook 8 anymore.
`:""}
      Please provide an explicit spy to your args like this:
        import { fn } from '@storybook/test';
        ... 
        args: {
         ${this.data.name}: fn()
        }
    `}};

export { Category, ImplicitActionsDuringRendering, MissingStoryAfterHmrError };

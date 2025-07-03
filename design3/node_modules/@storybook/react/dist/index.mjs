import { renderToCanvas, render } from './chunk-JWY6Y6NU.mjs';
import { global } from '@storybook/global';
import { start, setProjectAnnotations as setProjectAnnotations$1, composeStory as composeStory$1, composeStories as composeStories$1 } from '@storybook/preview-api';
import { deprecate } from '@storybook/client-logger';

var{window:globalWindow}=global;globalWindow&&(globalWindow.STORYBOOK_ENV="react");var RENDERER="react",api=start(renderToCanvas,{render}),storiesOf=(kind,m)=>api.clientApi.storiesOf(kind,m).addParameters({renderer:RENDERER}),configure=(...args)=>api.configure(RENDERER,...args),forceReRender=api.forceReRender,raw=api.clientApi.raw;function setProjectAnnotations(projectAnnotations){setProjectAnnotations$1(projectAnnotations);}function setGlobalConfig(projectAnnotations){deprecate("setGlobalConfig is deprecated. Use setProjectAnnotations instead."),setProjectAnnotations(projectAnnotations);}var defaultProjectAnnotations={render};function composeStory(story,componentAnnotations,projectAnnotations,exportsName){return composeStory$1(story,componentAnnotations,projectAnnotations,defaultProjectAnnotations,exportsName)}function composeStories(csfExports,projectAnnotations){return composeStories$1(csfExports,projectAnnotations,composeStory)}typeof module<"u"&&module?.hot?.decline();

export { composeStories, composeStory, configure, forceReRender, raw, setGlobalConfig, setProjectAnnotations, storiesOf };

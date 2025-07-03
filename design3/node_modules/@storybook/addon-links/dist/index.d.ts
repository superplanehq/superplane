import * as lib_preview_api_dist_addons from '@storybook/preview-api/dist/addons';
import { ComponentTitle, StoryName, StoryId, StoryKind } from '@storybook/types';

interface ParamsId {
    storyId: StoryId;
}
interface ParamsCombo {
    kind?: StoryKind;
    title?: ComponentTitle;
    story?: StoryName;
    name?: StoryName;
}
declare const navigate: (params: ParamsId | ParamsCombo) => void;
declare const hrefTo: (title: ComponentTitle, name: StoryName) => Promise<string>;
declare const linkTo: (idOrTitle: string | ((...args: any[]) => string), nameInput?: string | ((...args: any[]) => string) | undefined) => (...args: any[]) => void;
declare const withLinks: lib_preview_api_dist_addons.MakeDecoratorResult;

/**
 * @deprecated please import this specific function from @storybook/addon-links/react
 */
declare function LinkTo(): null;

export { LinkTo, hrefTo, linkTo, navigate, withLinks };

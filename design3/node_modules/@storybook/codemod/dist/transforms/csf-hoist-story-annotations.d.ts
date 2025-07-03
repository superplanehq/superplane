/**
 * Hoist CSF .story annotations
 *
 * For example:
 *
 * ```
 * export const Basic = () => <Button />
 * Basic.story = {
 *   name: 'foo',
 *   parameters: { ... },
 *   decorators = [ ... ],
 * };
 * ```
 *
 * Becomes:
 *
 * ```
 * export const Basic = () => <Button />
 * Basic.storyName = 'foo';
 * Basic.parameters = { ... };
 * Basic.decorators = [ ... ];
 * ```
 */
declare function transformer(file: any, api: any): any;

export { transformer as default };

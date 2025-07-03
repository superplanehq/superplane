/**
 * Takes the deprecated addon-info API, addWithInfo, and
 * converts to the new withInfo API.
 *
 * Example of deprecated addWithInfo API:
 *
 * storiesOf('Button')
 *   .addWithInfo(
 *     'story name',
 *     'Story description.',
 *     () => (
 *       <Button label="The Button" />
 *     )
 *   )
 *
 * Converts to the new withInfo API:
 *
 * storiesOf('Button')
 *   .add('story name', withInfo(
 *     'Story description.'
 *   )(() => (
 *     <Button label="The Button" />
 *   )))
 */
declare function transformer(file: any, api: any): any;

export { transformer as default };

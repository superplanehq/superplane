<h1 class="dn">
<a href="https://tscanlin.github.io/tocbot">Tocbot</a>
</h1>

<a class="no-decoration" href="https://github.com/tscanlin/tocbot/actions" target="_blank"><img src="https://github.com/tscanlin/tocbot/actions/workflows/node.js.yml/badge.svg" alt="github-actions" /></a>
<a class="no-decoration" href="https://cdnjs.com/libraries/tocbot" rel="nofollow"><img src="https://img.shields.io/cdnjs/v/tocbot" alt="cdnjs" style="max-width: 100%;"></a>
<a class="no-decoration" href="https://github.com/sponsors/tscanlin" rel="nofollow"><img alt="GitHub Sponsors" src="https://img.shields.io/github/sponsors/tscanlin"></a>

Tocbot builds a [table of contents](https://en.wikipedia.org/wiki/Table_of_contents) (TOC) from headings in an HTML document. This is useful for documentation websites or long markdown pages because it makes them easier to navigate. This library was inspired by [Tocify](http://gregfranko.com/jquery.tocify.js/), the main difference is that Tocbot uses native DOM methods and avoids the jQuery & jQuery UI dependencies.


## Get Started

You can use npm to install it or include the script on the page with HTML.

[**Download it here**](https://github.com/tscanlin/tocbot/releases/)


### Include JS

Install it with npm.

```sh
npm install --save tocbot
```

Then use with either commonjs or ESM imports:

```js
const tocbot = require('tocbot/dist/tocbot.js')
// OR
import tocbot from 'tocbot'

// Initialize tocbot
tocbot.init()
```

OR

Include the script at the bottom of the page before the closing body tag.

```html
<script src="https://cdnjs.cloudflare.com/ajax/libs/tocbot/4.32.2/tocbot.min.js"></script>
```


### Include CSS

CSS is used for expanding & collapsing groupings and some basic styling.

```html
<link rel="stylesheet" href="https://cdnjs.cloudflare.com/ajax/libs/tocbot/4.32.2/tocbot.css">
```

OR

If you installed it with npm and use sass / postcss you might try importing the styles from 'node_modules', [see the includePath option documentation for more info](https://github.com/sass/node-sass#includepaths)

```scss
@import 'tocbot/src/scss/tocbot';
```


### Usage

Initialize the script at the bottom of the page before the closing body tag.

```js
tocbot.init({
  // Where to render the table of contents.
  tocSelector: '.js-toc',
  // Where to grab the headings to build the table of contents.
  contentSelector: '.js-toc-content',
  // Which headings to grab inside of the contentSelector element.
  headingSelector: 'h1, h2, h3',
  // For headings inside relative or absolute positioned containers within content.
  hasInnerContainers: true,
});
```

**NOTE:** Make sure the body is scrollable and the document headings have id attributes, tocbot and your browser needs these things to make hashes jump to the proper heading. Some markdown libraries already do this for you: [marked](https://github.com/markedjs/marked) [can be configured to generate IDs based on patterns you define](https://github.com/markedjs/marked/issues/326#issuecomment-2567091770); [Showdown.js](https://showdownjs.com) generates IDs by default.

If content in the div has changed then trigger a refresh (optionally with new options).

```javascript
tocbot.refresh();
```

Also you can use it within typescript:

```typescript
import * as tocbot from 'tocbot';

tocbot.init({
  // Options
});

tocbot.refresh();

tocbot.destroy();
```

## Examples

- [Tocbot Homepage](https://tscanlin.github.io/tocbot/)
- [Storybook uses Tocbot](https://storybook.js.org/docs/writing-docs/autodocs#configure-the-table-of-contents)
  - Tocbot is used under the hood in storybook to provide TOC generation for component docs in storybook.

If you'd like to add your page to this list open a pull request.


## Requirements

This library uses **vanilla JavaScript**. It is less than 350 bytes of CSS and about 3.6Kb of JavaScript (minified and gzipped) it also has no dependencies.

This script works in **all modern browsers and IE 9+**.

**Make sure rendered headings have id attributes**, some markdown libraries (like [marked](https://github.com/chjj/marked)) already do this. If you need to do this client side see [this script](https://github.com/tscanlin/tocbot/blob/master/src/utils/make-ids.js).

**NOTE:** to exclude anchor elements from smooth scrolling, add the class `no-smooth-scroll`.


### Fixed headers

To handle fixed headers with tocbot, just pass the header offsets as options to tocbot. For example, the options needed for a `40px` tall fixed header would be:

```js
tocbot.init({
  headingsOffset: 40,
  scrollSmoothOffset: -40
})
```


## API

### Options

```javascript
// Where to render the table of contents.
  tocSelector: '.js-toc',
  // Or, you can pass in a DOM node instead
  tocElement: null,
  // Where to grab the headings to build the table of contents.
  contentSelector: '.js-toc-content',
  // Or, you can pass in a DOM node instead
  contentElement: null,
  // Which headings to grab inside of the contentSelector element.
  headingSelector: 'h1, h2, h3',
  // Headings that match the ignoreSelector will be skipped.
  ignoreSelector: '.js-toc-ignore',
  // For headings inside relative or absolute positioned
  // containers within content.
  hasInnerContainers: false,
  // Main class to add to links.
  linkClass: 'toc-link',
  // Extra classes to add to links.
  extraLinkClasses: '',
  // Class to add to active links,
  // the link corresponding to the top most heading on the page.
  activeLinkClass: 'is-active-link',
  // Main class to add to lists.
  listClass: 'toc-list',
  // Extra classes to add to lists.
  extraListClasses: '',
  // Class that gets added when a list should be collapsed.
  isCollapsedClass: 'is-collapsed',
  // Class that gets added when a list should be able
  // to be collapsed but isn't necessarily collapsed.
  collapsibleClass: 'is-collapsible',
  // Class to add to list items.
  listItemClass: 'toc-list-item',
  // Class to add to active list items.
  activeListItemClass: 'is-active-li',
  // How many heading levels should not be collapsed.
  // For example, number 6 will show everything since
  // there are only 6 heading levels and number 0 will collapse them all.
  // The sections that are hidden will open
  // and close as you scroll to headings within them.
  collapseDepth: 0,
  // Smooth scrolling enabled.
  scrollSmooth: true,
  // Smooth scroll duration.
  scrollSmoothDuration: 420,
  // Smooth scroll offset.
  scrollSmoothOffset: 0,
  // Callback for scroll end.
  scrollEndCallback: function (e) {},
  // Headings offset between the headings and the top of
  // the document (this is meant for minor adjustments).
  headingsOffset: 1,
  // Enable the URL hash to update with the proper heading ID as
  // a user scrolls the page.
  enableUrlHashUpdateOnScroll: false,
  // type of scroll handler to use. to make scroll event not too rapid.
  // Options are: "debounce" or "throttle"
  // when set auto , use debounce less than 333ms , other use throttle.
  // for ios browser can't use history.pushState() more than 100 times per 30 seconds reason
  scrollHandlerType: 'auto',
  //  scrollHandler delay in ms.
  scrollHandlerTimeout: 50,
  // Timeout between events firing to make sure it's
  // not too rapid (for performance reasons).
  throttleTimeout: 50,
  // Element to add the positionFixedClass to.
  positionFixedSelector: null,
  // Fixed position class to add to make sidebar fixed after scrolling
  // down past the fixedSidebarOffset.
  positionFixedClass: 'is-position-fixed',
  // fixedSidebarOffset can be any number but by default is set
  // to auto which sets the fixedSidebarOffset to the sidebar
  // element's offsetTop from the top of the document on init.
  fixedSidebarOffset: 'auto',
  // includeHtml can be set to true to include the HTML markup from the
  // heading node instead of just including the innerText.
  includeHtml: false,
  // includeTitleTags automatically sets the html title tag of the link
  // to match the title. This can be useful for SEO purposes or
  // when truncating titles.
  includeTitleTags: false,
  // onclick function to apply to all links in toc. will be called with
  // the event as the first parameter, and this can be used to stop,
  // propagation, prevent default or perform action
  onClick: function (e) {},
  // orderedList can be set to false to generate unordered lists (ul)
  // instead of ordered lists (ol)
  orderedList: true,
  // If there is a fixed article scroll container, set to calculate offset.
  scrollContainer: null,
  // prevent ToC DOM rendering if it's already rendered by an external system.
  skipRendering: false,
  // Optional callback to change heading labels.
  // For example it can be used to cut down and put ellipses on multiline headings you deem too long.
  // Called each time a heading is parsed. Expects a string and returns the modified label to display.
  // Additionally, the attribute `data-heading-label` may be used on a heading to specify
  // a shorter string to be used in the TOC.
  // function (string) => string
  headingLabelCallback: false,
  // ignore headings that are hidden in DOM
  ignoreHiddenElements: false,
  // Optional callback to modify properties of parsed headings.
  // The heading element is passed in node parameter and information
  // parsed by default parser is provided in obj parameter.
  // Function has to return the same or modified obj.
  // The heading will be excluded from TOC if nothing is returned.
  // function (object, HTMLElement) => object | void
  headingObjectCallback: null,
  // Set the base path, useful if you use a `base` tag in `head`.
  basePath: '',
  // Only takes affect when `tocSelector` is scrolling,
  // keep the toc scroll position in sync with the content.
  disableTocScrollSync: false,
  // If this is null then just use `tocElement` or `tocSelector` instead
  // assuming `disableTocScrollSync` is set to false. This allows for
  // scrolling an outer element (like a nav panel w/ search) containing the toc.
  // Please pass an element, not a selector here.
  tocScrollingWrapper: null,
  // Offset for the toc scroll (top) position when scrolling the page.
  // Only effective if `disableTocScrollSync` is false.
  tocScrollOffset: 30,
  // Threshold for when bottom mode should be enabled to handle
  // highlighting links that cannot be scrolled to.
  bottomModeThreshold: 30,
```


### Methods

#### .init

Initialize tocbot with an options object.

```javascript
tocbot.init(options)
```

#### .destroy

Destroy tocbot and remove event listeners.

```javascript
tocbot.destroy()
```

#### .refresh

Refresh tocbot if the document changes and it needs to be rebuilt.

```javascript
tocbot.refresh()
```


## Troubleshooting

- Tocbot scrolls to the right position onClick but highlighting doesn't seem to show the active section
  - Try running this from the console: `tocbot.refresh({ ...tocbot.options, hasInnerContainers: true })`. If that works then one option (`hasInnerContainers: true`) to handle inner containers should be all you need to add.
- If you have a really long TOC and are seeing headings getting truncated, then have a [look at this issue for a workaround to resolve it](https://github.com/tscanlin/tocbot/issues/330).


## Contributing

Contributions and suggestions are welcome! Please feel free to open an issue if you run into a problem or have a feature request. I'll do my best to respond in a timely fashion.

If you want to open a pull request just fork the repo but please make sure all tests and lint pass.


### Running Tests

```bash
npm run test
```

### Steps to publish

- Push a branch and open a pull request
- run `npm version <patch|minor|major>`
- Update readme.md with notes
- Merge the pull request
- commit dist/
- run `npm publish`
- make release on github


## Sponsors

If you find Tocbot useful, consider supporting it by becoming a sponsor. You can do this by visiting [the tocbot repo on github](https://github.com/tscanlin/tocbot) and clicking the "Sponsor" button at the top of the page. This will help me continue to develop and maintain Tocbot. Sponsors that contribute $5 /month or more will have their logos shown below. Thank you!

<img width="50" height="50" src="https://raw.githubusercontent.com/tscanlin/tocbot/master/static/img/getsentry.png" alt="Sentry"><span>&nbsp;</span><a href="https://sentry.io/" target="_blank">Sentry</a>
<br /><br />

<img width="50" height="50" src="https://raw.githubusercontent.com/tscanlin/tocbot/master/static/img/roboflow.png" alt="Roboflow"><span>&nbsp;</span><a href="https://www.roboflow.com/" target="_blank">Roboflow</a>
<br /><br />


## License

[MIT](http://opensource.org/licenses/MIT)

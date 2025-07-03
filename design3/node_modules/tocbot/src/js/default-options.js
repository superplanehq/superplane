export default {
  // Where to render the table of contents.
  tocSelector: ".js-toc",
  // Or, you can pass in a DOM node instead
  tocElement: null,
  // Where to grab the headings to build the table of contents.
  contentSelector: ".js-toc-content",
  // Or, you can pass in a DOM node instead
  contentElement: null,
  // Which headings to grab inside of the contentSelector element.
  headingSelector: "h1, h2, h3",
  // Headings that match the ignoreSelector will be skipped.
  ignoreSelector: ".js-toc-ignore",
  // For headings inside relative or absolute positioned
  // containers within content.
  hasInnerContainers: false,
  // Main class to add to links.
  linkClass: "toc-link",
  // Extra classes to add to links.
  extraLinkClasses: "",
  // Class to add to active links,
  // the link corresponding to the top most heading on the page.
  activeLinkClass: "is-active-link",
  // Main class to add to lists.
  listClass: "toc-list",
  // Extra classes to add to lists.
  extraListClasses: "",
  // Class that gets added when a list should be collapsed.
  isCollapsedClass: "is-collapsed",
  // Class that gets added when a list should be able
  // to be collapsed but isn't necessarily collapsed.
  collapsibleClass: "is-collapsible",
  // Class to add to list items.
  listItemClass: "toc-list-item",
  // Class to add to active list items.
  activeListItemClass: "is-active-li",
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
  scrollHandlerType: "auto",
  //  scrollHandler delay in ms.
  scrollHandlerTimeout: 50,
  // Timeout between events firing to make sure it's
  // not too rapid (for performance reasons).
  throttleTimeout: 50,
  // Element to add the positionFixedClass to.
  positionFixedSelector: null,
  // Fixed position class to add to make sidebar fixed after scrolling
  // down past the fixedSidebarOffset.
  positionFixedClass: "is-position-fixed",
  // fixedSidebarOffset can be any number but by default is set
  // to auto which sets the fixedSidebarOffset to the sidebar
  // element's offsetTop from the top of the document on init.
  fixedSidebarOffset: "auto",
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
  basePath: "",
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
}

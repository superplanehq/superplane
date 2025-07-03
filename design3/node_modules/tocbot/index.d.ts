export default tocbot
export as namespace tocbot

declare namespace tocbot {
  /**
   * @see https://github.com/tscanlin/tocbot#options
   */
  interface IStaticOptions {
    // Where to render the table of contents.
    tocSelector?: string
    // Or, you can pass in a DOM node instead
    tocElement?: HTMLElement
    // Where to grab the headings to build the table of contents.
    contentSelector?: string
    // Or, you can pass in a DOM node instead
    contentElement?: HTMLElement
    // Which headings to grab inside of the contentSelector element.
    headingSelector?: string
    // Headings that match the ignoreSelector will be skipped.
    ignoreSelector?: string
    // For headings inside relative or absolute positioned
    // containers within content.
    hasInnerContainers?: boolean
    // Main class to add to links.
    linkClass?: string
    // Extra classes to add to links.
    extraLinkClasses?: string
    // Class to add to active links,
    // the link corresponding to the top most heading on the page.
    activeLinkClass?: string
    // Main class to add to lists.
    listClass?: string
    // Extra classes to add to lists.
    extraListClasses?: string
    // Class that gets added when a list should be collapsed.
    isCollapsedClass?: string
    // Class that gets added when a list should be able
    // to be collapsed but isn't necessarily collapsed.
    collapsibleClass?: string
    // Class to add to list items.
    listItemClass?: string
    // Class to add to active list items.
    activeListItemClass?: string
    // How many heading levels should not be collapsed.
    // For example, number 6 will show everything since
    // there are only 6 heading levels and number 0 will collapse them all.
    // The sections that are hidden will open
    // and close as you scroll to headings within them.
    collapseDepth?: number
    // Smooth scrolling enabled.
    scrollSmooth?: boolean
    // Smooth scroll duration.
    scrollSmoothDuration?: number
    // Smooth scroll offset.
    scrollSmoothOffset?: number
    // Callback for scroll end.
    scrollEndCallback?(e: WheelEvent): void
    // Headings offset between the headings and the top of
    // the document (this is meant for minor adjustments).
    headingsOffset?: number
    // Enable the URL hash to update with the proper heading ID as
    // a user scrolls the page.
    enableUrlHashUpdateOnScroll?: boolean
    // type of scroll handler to use. to make scroll event not too rapid.
    // Options are: "debounce" or "throttle"
    // when set auto , use debounce less than 333ms , other use throttle.
    // for ios browser can't use history.pushState() more than 100 times per 30 seconds reason
    scrollHandlerType?: "auto" | "debounce" | "throttle"
    //  scrollHandler delay in ms.
    scrollHandlerTimeout?: number
    // Timeout between events firing to make sure it's
    // not too rapid (for performance reasons).
    throttleTimeout?: number
    // Element to add the positionFixedClass to.
    positionFixedSelector?: string | null
    // Fixed position class to add to make sidebar fixed after scrolling
    // down past the fixedSidebarOffset.
    positionFixedClass?: string
    // fixedSidebarOffset can be any number but by default is set
    // to auto which sets the fixedSidebarOffset to the sidebar
    // element's offsetTop from the top of the document on init.
    fixedSidebarOffset?: "auto" | number
    // includeHtml can be set to true to include the HTML markup from the
    // heading node instead of just including the innerText.
    includeHtml?: boolean
    // includeTitleTags automatically sets the html title tag of the link
    // to match the title. This can be useful for SEO purposes or
    // when truncating titles.
    includeTitleTags?: boolean
    // onclick function to apply to all links in toc. will be called with
    // the event as the first parameter, and this can be used to stop,
    // propagation, prevent default or perform action
    onClick?: (e: MouseEvent) => void
    // orderedList can be set to false to generate unordered lists (ul)
    // instead of ordered lists (ol)
    orderedList?: boolean
    // If there is a fixed article scroll container, set to calculate offset.
    scrollContainer?: string | null
    // prevent ToC DOM rendering if it's already rendered by an external system.
    skipRendering?: boolean
    // Optional callback to change heading labels.
    // For example it can be used to cut down and put ellipses on multiline headings you deem too long.
    // Called each time a heading is parsed. Expects a string and returns the modified label to display.
    // Additionally, the attribute `data-heading-label` may be used on a heading to specify
    // a shorter string to be used in the TOC.
    headingLabelCallback?: (headingLabel: string) => string
    // ignore headings that are hidden in DOM
    ignoreHiddenElements?: boolean
    // Optional callback to modify properties of parsed headings.
    // The heading element is passed in node parameter and information
    // parsed by default parser is provided in obj parameter.
    // Function has to return the same or modified obj.
    // The heading will be excluded from TOC if nothing is returned.
    headingObjectCallback?: (obj: object, node: HTMLElement) => object | void
    // Set the base path, useful if you use a `base` tag in `head`.
    basePath?: string
    // Only takes affect when `tocSelector` is scrolling,
    // keep the toc scroll position in sync with the content.
    disableTocScrollSync?: boolean
    // If this is null then just use `tocElement` or `tocSelector` instead
    // assuming `disableTocScrollSync` is set to false. This allows for
    // scrolling an outer element (like a nav panel w/ search) containing the toc.
    // Please pass an element, not a selector here.
    tocScrollingWrapper?: null
    // Offset for the toc scroll (top) position when scrolling the page.
    // Only effective if `disableTocScrollSync` is false.
    tocScrollOffset?: number
    // Threshold for when bottom mode should be enabled to handle
    // highlighting links that cannot be scrolled to.
    bottomModeThreshold?: number
  }

  /**
   * Initialize tocbot with an options object.
   * @see https://github.com/tscanlin/tocbot#init
   */
  function init(options?: IStaticOptions): void
  /**
   * Destroy tocbot and remove event listeners.
   * @see https://github.com/tscanlin/tocbot#destroy
   */
  function destroy(): void
  /**
   * Refresh tocbot if the document changes and it needs to be rebuilt.
   * @see https://github.com/tscanlin/tocbot#refresh
   */
  function refresh(options?: IStaticOptions): void
}

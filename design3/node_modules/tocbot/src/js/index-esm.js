/* eslint no-var: off */
/**
 * Tocbot
 * Tocbot creates a table of contents based on HTML headings on a page,
 * this allows users to easily jump to different sections of the document.
 * Tocbot was inspired by tocify (http://gregfranko.com/jquery.tocify.js/).
 * The main differences are that it works natively without any need for jquery or jquery UI).
 *
 * @author Tim Scanlin
 */

import BuildHtml from "./build-html.js"
import defaultOptions from "./default-options.js"
import ParseContent from "./parse-content.js"
import initSmoothScrolling from "./scroll-smooth/index.js"
import updateTocScroll from "./update-toc-scroll.js"

// For testing purposes.
export let _options = {} // Object to store current options.
export let _buildHtml
export let _parseContent
export let _headingsArray
export let _scrollListener

let clickListener

/**
 * Initialize tocbot.
 * @param {object} customOptions
 */
export function init(customOptions) {
  // Merge defaults with user options.
  // Set to options variable at the top.
  let hasInitialized = false
  _options = extend(defaultOptions, customOptions || {})

  // Init smooth scroll if enabled (default).
  if (_options.scrollSmooth) {
    _options.duration = _options.scrollSmoothDuration
    _options.offset = _options.scrollSmoothOffset

    initSmoothScrolling(_options)
  }

  // Pass options to these modules.
  _buildHtml = BuildHtml(_options)
  _parseContent = ParseContent(_options)

  // Destroy it if it exists first.
  destroy()

  const contentElement = getContentElement(_options)
  if (contentElement === null) {
    return
  }

  const tocElement = getTocElement(_options)
  if (tocElement === null) {
    return
  }

  // Get headings array.
  _headingsArray = _parseContent.selectHeadings(
    contentElement,
    _options.headingSelector,
  )

  // Return if no headings are found.
  if (_headingsArray === null) {
    return
  }

  // Build nested headings array.
  const nestedHeadingsObj = _parseContent.nestHeadingsArray(_headingsArray)
  const nestedHeadings = nestedHeadingsObj.nest

  // Render.
  if (!_options.skipRendering) {
    _buildHtml.render(tocElement, nestedHeadings)
  } else {
    // No need to attach listeners if skipRendering is true, this was causing errors.
    return this
  }

  // Update Sidebar and bind listeners.
  let isClick = false
  // choose timeout by _options
  const scrollHandlerTimeout =
    _options.scrollHandlerTimeout || _options.throttleTimeout // compatible with legacy configurations
  // choose debounce or throttle
  // default use debounce when delay is less than 333ms
  // the reason is ios browser has a limit : can't use history.pushState() more than 100 times per 30 seconds
  const scrollHandler = (fn, delay) =>
    getScrollHandler(fn, delay, _options.scrollHandlerType)

  _scrollListener = scrollHandler((e) => {
    _buildHtml.updateToc(_headingsArray, e)
    // Only do this update for normal scrolls and not during clicks.
    !_options.disableTocScrollSync && !isClick && updateTocScroll(_options)

    if (_options.enableUrlHashUpdateOnScroll && hasInitialized) {
      const enableUpdatingHash = _buildHtml.getCurrentlyHighlighting()
      enableUpdatingHash && _buildHtml.updateUrlHashForHeader(_headingsArray)
    }

    const isTop = e?.target?.scrollingElement?.scrollTop === 0
    if ((e && (e.eventPhase === 0 || e.currentTarget === null)) || isTop) {
      _buildHtml.updateToc(_headingsArray)
      _options.scrollEndCallback?.(e)
    }
  }, scrollHandlerTimeout)
  // Fire it initially to setup the page.
  if (!hasInitialized) {
    _scrollListener()
    hasInitialized = true
  }

  // Fire scroll listener on hash change to trigger highlighting changes too.
  window.onhashchange = window.onscrollend = (e) => {
    _scrollListener(e)
  }

  if (
    _options.scrollContainer &&
    document.querySelector(_options.scrollContainer)
  ) {
    document
      .querySelector(_options.scrollContainer)
      .addEventListener("scroll", _scrollListener, false)
    document
      .querySelector(_options.scrollContainer)
      .addEventListener("resize", _scrollListener, false)
  } else {
    document.addEventListener("scroll", _scrollListener, false)
    document.addEventListener("resize", _scrollListener, false)
  }

  // Bind click listeners to disable animation.
  let timeout = null
  clickListener = throttle((event) => {
    isClick = true
    if (_options.scrollSmooth) {
      _buildHtml.disableTocAnimation(event)
    }
    _buildHtml.updateToc(_headingsArray, event)
    // Timeout to re-enable the animation.
    timeout && clearTimeout(timeout)
    timeout = setTimeout(() => {
      _buildHtml.enableTocAnimation()
    }, _options.scrollSmoothDuration)
    // Set is click w/ a bit of delay so that animations can finish
    // and we don't disturb the user while they click the toc.
    setTimeout(() => {
      isClick = false
    }, _options.scrollSmoothDuration + 100)
  }, _options.throttleTimeout)

  if (
    _options.scrollContainer &&
    document.querySelector(_options.scrollContainer)
  ) {
    document
      .querySelector(_options.scrollContainer)
      .addEventListener("click", clickListener, false)
  } else {
    document.addEventListener("click", clickListener, false)
  }
}

/**
 * Destroy tocbot.
 */
export function destroy() {
  const tocElement = getTocElement(_options)
  if (tocElement === null) {
    return
  }

  if (!_options.skipRendering) {
    // Clear HTML.
    if (tocElement) {
      tocElement.innerHTML = ""
    }
  }

  // Remove event listeners.
  if (
    _options.scrollContainer &&
    document.querySelector(_options.scrollContainer)
  ) {
    document
      .querySelector(_options.scrollContainer)
      .removeEventListener("scroll", _scrollListener, false)
    document
      .querySelector(_options.scrollContainer)
      .removeEventListener("resize", _scrollListener, false)
    if (_buildHtml) {
      document
        .querySelector(_options.scrollContainer)
        .removeEventListener("click", clickListener, false)
    }
  } else {
    document.removeEventListener("scroll", _scrollListener, false)
    document.removeEventListener("resize", _scrollListener, false)
    if (_buildHtml) {
      document.removeEventListener("click", clickListener, false)
    }
  }
}

/**
 * Refresh tocbot.
 */
export function refresh(customOptions) {
  destroy()
  init(customOptions || _options)
}

// From: https://github.com/Raynos/xtend
const hasOwnProp = Object.prototype.hasOwnProperty
function extend(...args) {
  const target = {}
  for (let i = 0; i < args.length; i++) {
    const source = args[i]
    for (const key in source) {
      if (hasOwnProp.call(source, key)) {
        target[key] = source[key]
      }
    }
  }
  return target
}

// From: https://remysharp.com/2010/07/21/throttling-function-calls
function throttle(fn, threshold, scope) {
  threshold || (threshold = 250)
  let last
  let deferTimer
  return function (...args) {
    const context = scope || this
    const now = +new Date()
    if (last && now < last + threshold) {
      // hold on to it
      clearTimeout(deferTimer)
      deferTimer = setTimeout(() => {
        last = now
        fn.apply(context, args)
      }, threshold)
    } else {
      last = now
      fn.apply(context, args)
    }
  }
}

/**
 * Creates a debounced function that delays invoking `func` until after `wait` milliseconds
 * have elapsed since the last time the debounced function was invoked.
 *
 * @param {Function} func - The function to debounce.
 * @param {number} wait - The number of milliseconds to delay.
 * @returns {Function} - Returns the new debounced function.
 */
function debounce(func, wait) {
  let timeout
  return (...args) => {
    clearTimeout(timeout)
    timeout = setTimeout(() => func.apply(this, args), wait)
  }
}

/**
 * Creates a scroll handler with specified timeout strategy
 * @param {number} timeout - Delay duration in milliseconds
 * @param {'debounce'|'throttle'|'auto'} type - Strategy type for scroll handling
 * @returns {Function} Configured scroll handler function
 */
function getScrollHandler(func, timeout, type = "auto") {
  switch (type) {
    case "debounce":
      return debounce(func, timeout)
    case "throttle":
      return throttle(func, timeout)
    default:
      return timeout < 334 ? debounce(func, timeout) : throttle(func, timeout)
  }
}

function getContentElement(options) {
  try {
    return (
      options.contentElement || document.querySelector(options.contentSelector)
    )
  } catch (e) {
    console.warn(`Contents element not found: ${options.contentSelector}`) // eslint-disable-line
    return null
  }
}

function getTocElement(options) {
  try {
    return options.tocElement || document.querySelector(options.tocSelector)
  } catch (e) {
    console.warn(`TOC element not found: ${options.tocSelector}`) // eslint-disable-line
    return null
  }
}

// Add default export for easier use.
const tocbot = {
  _options,
  _buildHtml,
  _parseContent,
  init,
  destroy,
  refresh,
}

export default tocbot

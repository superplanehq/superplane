/**
 * This file is responsible for parsing the content from the DOM and making
 * sure data is nested properly.
 *
 * @author Tim Scanlin
 */

export default function parseContent(options) {
  const reduce = [].reduce

  /**
   * Get the last item in an array and return a reference to it.
   * @param {Array} array
   * @return {Object}
   */
  function getLastItem(array) {
    return array[array.length - 1]
  }

  /**
   * Get heading level for a heading dom node.
   * @param {HTMLElement} heading
   * @return {Number}
   */
  function getHeadingLevel(heading) {
    return +heading.nodeName.toUpperCase().replace("H", "")
  }

  /**
   * Determine whether the object is an HTML Element.
   * Also works inside iframes. HTML Elements might be created by the parent document.
   * @param {Object} maybeElement
   * @return {Number}
   */
  function isHTMLElement(maybeElement) {
    try {
      return (
        maybeElement instanceof window.HTMLElement ||
        maybeElement instanceof window.parent.HTMLElement
      )
    } catch (e) {
      return maybeElement instanceof window.HTMLElement
    }
  }

  /**
   * Get important properties from a heading element and store in a plain object.
   * @param {HTMLElement} heading
   * @return {Object}
   */
  function getHeadingObject(heading) {
    // each node is processed twice by this method because nestHeadingsArray() and addNode() calls it
    // first time heading is real DOM node element, second time it is obj
    // that is causing problem so I am processing only original DOM node
    if (!isHTMLElement(heading)) return heading

    if (
      options.ignoreHiddenElements &&
      (!heading.offsetHeight || !heading.offsetParent)
    ) {
      return null
    }

    const headingLabel =
      heading.getAttribute("data-heading-label") ||
      (options.headingLabelCallback
        ? String(options.headingLabelCallback(heading.innerText))
        : (heading.innerText || heading.textContent).trim())
    const obj = {
      id: heading.id,
      children: [],
      nodeName: heading.nodeName,
      headingLevel: getHeadingLevel(heading),
      textContent: headingLabel,
    }

    if (options.includeHtml) {
      obj.childNodes = heading.childNodes
    }

    if (options.headingObjectCallback) {
      return options.headingObjectCallback(obj, heading)
    }

    return obj
  }

  /**
   * Add a node to the nested array.
   * @param {Object} node
   * @param {Array} nest
   * @return {Array}
   */
  function addNode(node, nest) {
    const obj = getHeadingObject(node)
    const level = obj.headingLevel
    let array = nest
    let lastItem = getLastItem(array)
    const lastItemLevel = lastItem ? lastItem.headingLevel : 0
    let counter = level - lastItemLevel

    while (counter > 0) {
      lastItem = getLastItem(array)
      // Handle case where there are multiple h5+ in a row.
      if (lastItem && level === lastItem.headingLevel) {
        break
      } else if (lastItem && lastItem.children !== undefined) {
        array = lastItem.children
      }
      counter--
    }

    if (level >= options.collapseDepth) {
      obj.isCollapsed = true
    }

    array.push(obj)
    return array
  }

  /**
   * Select headings in content area, exclude any selector in options.ignoreSelector
   * @param {HTMLElement} contentElement
   * @param {Array} headingSelector
   * @return {Array}
   */
  function selectHeadings(contentElement, headingSelector) {
    let selectors = headingSelector
    if (options.ignoreSelector) {
      selectors = headingSelector
        .split(",")
        .map(function mapSelectors(selector) {
          return `${selector.trim()}:not(${options.ignoreSelector})`
        })
    }
    try {
      return contentElement.querySelectorAll(selectors)
    } catch (e) {
      console.warn(`Headers not found with selector: ${selectors}`) // eslint-disable-line
      return null
    }
  }

  /**
   * Nest headings array into nested arrays with 'children' property.
   * @param {Array} headingsArray
   * @return {Object}
   */
  function nestHeadingsArray(headingsArray) {
    return reduce.call(
      headingsArray,
      function reducer(prev, curr) {
        const currentHeading = getHeadingObject(curr)
        if (currentHeading) {
          addNode(currentHeading, prev.nest)
        }
        return prev
      },
      {
        nest: [],
      },
    )
  }

  return {
    nestHeadingsArray,
    selectHeadings,
  }
}

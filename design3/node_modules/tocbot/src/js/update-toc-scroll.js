export default function updateTocScroll(options) {
  const toc =
    options.tocScrollingWrapper ||
    options.tocElement ||
    document.querySelector(options.tocSelector)
  if (toc && toc.scrollHeight > toc.clientHeight) {
    const activeItem = toc.querySelector(`.${options.activeListItemClass}`)
    if (activeItem) {
      // Determine element top and bottom
      const eTop = activeItem.offsetTop

      // Check if out of view
      // Above scroll view
      const scrollAmount = eTop - options.tocScrollOffset
      toc.scrollTop = scrollAmount > 0 ? scrollAmount : 0
    }
  }
}

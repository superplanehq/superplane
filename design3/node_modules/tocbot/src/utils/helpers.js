export function isElementInViewport(el) {
  const rect = el.getBoundingClientRect()
  return (
    rect.top >= 0 &&
    rect.left >= 0 &&
    rect.bottom <=
      (window.innerHeight || document.documentElement.clientHeight) &&
    rect.right <= (window.innerWidth || document.documentElement.clientWidth)
  )
}

export function updateUrlHashForHeader(headingsArray) {
  if (typeof window === "undefined") return
  const body = document.body
  const scrollTop = document.documentElement.scrollTop || body.scrollTop

  let hasElInView = false
  for (const el of headingsArray) {
    if (isElementInViewport(el) && !hasElInView) {
      const newHash = `#${el.id}`
      if (window.location.hash !== newHash) {
        window.history.pushState(null, null, newHash)
      }
      hasElInView = true
    }
  }
  if (scrollTop === 0 || (scrollTop < 4 && !hasElInView)) {
    //! hasElInView) {
    console.log("a: ", window.location.hash === "")
    if (!(window.location.hash === "#" || window.location.hash === "")) {
      window.history.pushState(null, null, "#")
    }
  }
}

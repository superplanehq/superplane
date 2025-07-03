function debounce(fn, delay) {
  let handle
  return () => {
    clearTimeout(handle)
    handle = setTimeout(fn, delay)
  }
}

/* eslint-disable no-undef */
const enqueueUpdate = debounce(exports.performReactRefresh, 16)

// Taken from https://github.com/pmmmwh/react-refresh-webpack-plugin/blob/main/lib/runtime/RefreshUtils.js#L141
// This allows to resister components not detected by SWC like styled component
function registerExportsForReactRefresh(filename, moduleExports) {
  for (const key in moduleExports) {
    if (key === '__esModule') continue
    const exportValue = moduleExports[key]
    if (exports.isLikelyComponentType(exportValue)) {
      exports.register(exportValue, filename + ' ' + key)
    }
  }
}

function validateRefreshBoundaryAndEnqueueUpdate(prevExports, nextExports) {
  if (!predicateOnExport(prevExports, (key) => !!nextExports[key])) {
    return 'Could not Fast Refresh (export removed)'
  }

  let hasExports = false
  const allExportsAreComponentsOrUnchanged = predicateOnExport(
    nextExports,
    (key, value) => {
      hasExports = true
      if (exports.isLikelyComponentType(value)) return true
      if (!prevExports[key]) return false
      return prevExports[key] === nextExports[key]
    },
  )
  if (hasExports && allExportsAreComponentsOrUnchanged) {
    enqueueUpdate()
  } else {
    return 'Could not Fast Refresh. Learn more at https://github.com/vitejs/vite-plugin-react/tree/main/packages/plugin-react#consistent-components-exports'
  }
}

function predicateOnExport(moduleExports, predicate) {
  for (const key in moduleExports) {
    if (key === '__esModule') continue
    const desc = Object.getOwnPropertyDescriptor(moduleExports, key)
    if (desc && desc.get) return false
    if (!predicate(key, moduleExports[key])) return false
  }
  return true
}

exports.registerExportsForReactRefresh = registerExportsForReactRefresh
exports.validateRefreshBoundaryAndEnqueueUpdate =
  validateRefreshBoundaryAndEnqueueUpdate

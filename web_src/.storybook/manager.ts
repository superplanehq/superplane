const params = new URLSearchParams(window.location.search);
if (!params.has("path") && !params.has("id")) {
  window.history.replaceState(null, "", "?path=/story/factories-pages-lines--populated");
}

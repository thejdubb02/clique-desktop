/* Links belong in the browser this person already uses, with their bookmarks,
   their extensions and their logins, not in a window that cannot go back.

   WebView2 raises NewWindowRequested for both of the ways the panel opens one,
   and go-webview2 does not surface that event, so a click did nothing at all.
   Both ways are caught here instead. */
(function () {
  var isExternal = function (u) { return /^https?:\/\//i.test(String(u || "")); };

  var open = window.open;
  window.open = function (u) {
    if (isExternal(u)) {
      window.cliqueOpenExternal(String(u));
      return null;
    }
    return open.apply(window, arguments);
  };

  /* Capturing, so the panel's own handlers cannot swallow it first. Same-origin
     links without target=_blank are the panel navigating itself and are left
     alone. */
  window.addEventListener("click", function (ev) {
    var a = ev.target && ev.target.closest ? ev.target.closest("a[href]") : null;
    if (!a || !isExternal(a.href)) return;
    if (a.target !== "_blank" && a.origin === location.origin) return;
    ev.preventDefault();
    window.cliqueOpenExternal(a.href);
  }, true);
})();

/* The panel picks its own colours. The Windows caption above it does not, so
   the two read as separate applications unless we copy --panel and --fg up.

   Theme changes land either as properties on :root or as a swapped style
   element in head, which is why both are watched. */
(function () {
  window.cssColorToInt = function (text) {
    if (text == null) return null;
    var s = String(text).trim();
    var m = /^#([0-9a-fA-F]{3})$/.exec(s);
    if (m) {
      var h = m[1];
      return parseInt(h.charAt(0) + h.charAt(0) + h.charAt(1) + h.charAt(1) + h.charAt(2) + h.charAt(2), 16);
    }
    m = /^#([0-9a-fA-F]{6})$/.exec(s);
    if (m) return parseInt(m[1], 16);
    m = /^rgba?\(\s*(\d+)\s*,\s*(\d+)\s*,\s*(\d+)/i.exec(s);
    if (m) {
      var r = Number(m[1]), g = Number(m[2]), b = Number(m[3]);
      if (r > 255 || g > 255 || b > 255) return null;
      return (r << 16) | (g << 8) | b;
    }
    return null;
  };

  var lastPanel = null;
  var lastFg = null;

  var send = function () {
    if (typeof window.cliqueCaption !== "function") return;
    var cs = getComputedStyle(document.documentElement);
    var panel = window.cssColorToInt(cs.getPropertyValue("--panel"));
    var fg = window.cssColorToInt(cs.getPropertyValue("--fg"));
    if (panel === null || fg === null) return;
    if (panel === lastPanel && fg === lastFg) return;
    lastPanel = panel;
    lastFg = fg;
    window.cliqueCaption(panel, fg);
  };

  var watch = function () {
    send();
    new MutationObserver(send).observe(document.documentElement, { attributes: true });
    if (document.head) {
      new MutationObserver(send).observe(document.head, { childList: true });
    }
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", watch);
  } else {
    watch();
  }
})();

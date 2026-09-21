(function () {
  /* Drawn by the panel itself, next to the version it updates (see
   * appendUpdateBadge in clique/web/app.js) — this file only sets state and
   * asks the panel to repaint, it does not build any DOM of its own. That
   * used to be a separately positioned floating button; moving it here means
   * it survives every poll's repaint instead of needing to reposition itself
   * independently of the version footer beside it. */
  window.__cliqueUpdate = function (version) {
    window.cliqueUpdateBadge = { version: version, state: "ready" };
    if (typeof renderVersion === "function") renderVersion();
  };

  window.__cliqueUpdateFailed = function (msg) {
    const badge = window.cliqueUpdateBadge;
    if (!badge) return;   // nothing staged, nothing to mark failed
    badge.state = "failed";
    badge.message = msg;
    if (typeof renderVersion === "function") renderVersion();
  };

  function showPending() {
    if (typeof window.cliqueUpdatePending !== "function") return;
    Promise.resolve(window.cliqueUpdatePending()).then(function (v) {
      if (v) window.__cliqueUpdate(v);
    });
  }

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", showPending);
  } else {
    showPending();
  }
})();

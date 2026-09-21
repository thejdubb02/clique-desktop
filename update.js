(function () {
  const ROOT_ID = "clique-update-root";

  /* The card sits inside the panel's page, so it can read the panel's own
     theme rather than being told it. The fallbacks are the old fixed colours,
     for the first-run setup page, which has no theme to read. */
  function themed(name, fallback) {
    try {
      var v = getComputedStyle(document.documentElement).getPropertyValue(name);
      v = (v || "").trim();
      return v || fallback;
    } catch (e) {
      return fallback;
    }
  }

  const ICON_UPDATE =
    '<path d="M12 4v10m0 0l-4-4m4 4l4-4M6 18h12" stroke="currentColor" stroke-width="2" ' +
    'stroke-linecap="round" stroke-linejoin="round" fill="none"/>';
  const ICON_SPIN =
    '<circle cx="12" cy="12" r="8" stroke="currentColor" stroke-width="2" fill="none" ' +
    'stroke-dasharray="34 12" stroke-linecap="round"/>';
  const ICON_ERR =
    '<path d="M12 8v5M12 16.5v.01" stroke="currentColor" stroke-width="2.2" ' +
    'stroke-linecap="round" fill="none"/>';

  /* One small round button, not a card: a click installs and restarts, no
   * second confirmation, so the only thing worth showing is which state it
   * is in — waiting for a click, working, or failed and clickable again. */
  window.__cliqueUpdate = function (version) {
    var cAccent = themed("--accent", "#0078d4");
    var cOnAccent = themed("--on-accent", "#ffffff");
    var cErr = "#f85149";
    const existing = document.getElementById(ROOT_ID);
    if (existing) existing.remove();

    const style = document.createElement("style");
    style.id = "clique-update-style";
    style.textContent =
      "#" + ROOT_ID + "{opacity:0;transform:scale(.7);transition:opacity 160ms ease,transform 160ms ease;}" +
      "#" + ROOT_ID + ".clique-update-in{opacity:1;transform:none;}" +
      "#" + ROOT_ID + "::before{content:'';position:absolute;inset:-4px;border-radius:50%;" +
      "box-shadow:0 0 0 0 var(--clique-update-ring);animation:clique-update-pulse 2.2s ease-out infinite;}" +
      "@keyframes clique-update-pulse{0%{box-shadow:0 0 0 0 var(--clique-update-ring);}" +
      "70%{box-shadow:0 0 0 9px transparent;}100%{box-shadow:0 0 0 0 transparent;}}" +
      "#" + ROOT_ID + " svg{animation:none;}" +
      "#" + ROOT_ID + ".clique-update-busy svg{animation:clique-update-spin 0.9s linear infinite;}" +
      "@keyframes clique-update-spin{to{transform:rotate(360deg);}}" +
      "@media (prefers-reduced-motion:reduce){#" + ROOT_ID + "::before{animation:none;box-shadow:none;}" +
      "#" + ROOT_ID + ",#" + ROOT_ID + ".clique-update-in{transition:none;}}";
    document.head.appendChild(style);

    const btn = document.createElement("button");
    btn.id = ROOT_ID;
    btn.type = "button";
    btn.title = "CLIque " + version + " is ready — click to install and restart";
    btn.setAttribute("aria-label", btn.title);
    btn.style.cssText = [
      "position:fixed", "right:20px", "bottom:20px", "z-index:2147483000",
      "width:40px", "height:40px", "border-radius:50%", "padding:0",
      "display:flex", "align-items:center", "justify-content:center",
      "background:" + cAccent, "color:" + cOnAccent, "border:none", "cursor:pointer",
      "box-shadow:0 4px 16px rgba(0,0,0,0.35)",
      "--clique-update-ring:" + cAccent + "99"
    ].join(";");

    const icon = document.createElementNS("http://www.w3.org/2000/svg", "svg");
    icon.setAttribute("width", "20");
    icon.setAttribute("height", "20");
    icon.setAttribute("viewBox", "0 0 24 24");
    icon.style.cssText = "transform-origin:center;display:block;";
    icon.innerHTML = ICON_UPDATE;
    btn.appendChild(icon);

    function setBusy() {
      btn.classList.add("clique-update-busy");
      btn.disabled = true;
      btn.title = "Installing CLIque " + version + "…";
      btn.setAttribute("aria-label", btn.title);
      icon.innerHTML = ICON_SPIN;
    }

    function setFailed(msg) {
      btn.classList.remove("clique-update-busy");
      btn.disabled = false;
      btn.style.background = cErr;
      btn.title = "Update failed: " + (msg || "unknown error") + " — click to retry";
      btn.setAttribute("aria-label", btn.title);
      icon.innerHTML = ICON_ERR;
    }

    window.__cliqueUpdateFailed = function (msg) {
      setFailed(msg);
    };

    btn.addEventListener("click", async function () {
      setBusy();
      try {
        const msg = await window.cliqueRestart();
        if (msg) setFailed(msg);
        // No message means the app is already on its way down to relaunch;
        // nothing left here should still be running to update.
      } catch (e) {
        setFailed(e && e.message ? e.message : String(e));
      }
    });

    (document.body || document.documentElement).appendChild(btn);
    requestAnimationFrame(function () {
      btn.classList.add("clique-update-in");
    });
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

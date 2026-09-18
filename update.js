(function () {
  const ROOT_ID = "clique-update-root";

  window.__cliqueUpdateFailed = function (msg) {
    const err = document.getElementById("clique-update-err");
    const restart = document.getElementById("clique-update-restart");
    const later = document.getElementById("clique-update-later");
    if (err) err.textContent = msg || "update failed";
    if (restart) {
      restart.disabled = false;
      restart.textContent = "Restart now";
    }
    if (later) later.disabled = false;
  };

  // Whether this copy was installed as a package. The two say different things:
  // an installed copy is updated by Windows and only needs restarting, a loose
  // exe downloads and swaps itself.
  var packaged = false;
  window.__cliqueUpdate = function (version, isPackaged) {
    packaged = !!isPackaged;
    const existing = document.getElementById(ROOT_ID);
    if (existing) existing.remove();

    const root = document.createElement("div");
    root.id = ROOT_ID;
    root.setAttribute("role", "status");
    root.style.cssText = [
      "position:fixed",
      "right:20px",
      "bottom:20px",
      "z-index:2147483000",
      "width:min(320px,92vw)",
      "box-sizing:border-box",
      "padding:16px 16px 14px",
      "background:#1e1e1e",
      "color:#e6e6e6",
      "border:1px solid #3a3a3a",
      "border-radius:12px",
      "box-shadow:0 12px 40px rgba(0,0,0,0.45)",
      "font:14px/1.45 'Segoe UI',system-ui,sans-serif",
      "letter-spacing:normal",
      "text-align:left"
    ].join(";");

    const style = document.createElement("style");
    style.id = "clique-update-style";
    style.textContent =
      "#" + ROOT_ID + "{opacity:0;transform:translateY(8px);transition:opacity 160ms ease,transform 160ms ease;}" +
      "#" + ROOT_ID + ".clique-update-in{opacity:1;transform:none;}" +
      "@media (prefers-reduced-motion: reduce){" +
      "#" + ROOT_ID + ",#" + ROOT_ID + ".clique-update-in{opacity:1;transform:none;transition:none;}" +
      "}";
    root.appendChild(style);

    const heading = document.createElement("div");
    heading.id = "clique-update-heading";
    heading.style.cssText = "font:600 15px/1.3 'Segoe UI',system-ui,sans-serif;margin:0 0 8px;color:#f3f3f3;";
    heading.textContent = "CLIque " + version + " is ready";
    root.appendChild(heading);

    const body = document.createElement("div");
    body.id = "clique-update-body";
    body.style.cssText = "font:13px/1.45 'Segoe UI',system-ui,sans-serif;margin:0 0 12px;color:#b3b3b3;";
    body.textContent = packaged
      ? "Windows installs it in the background. Restart when you like to pick it up. Your sessions keep running, they live on the server, so nothing is lost."
      : "Restart when you like. Your sessions keep running, they live on the server, so nothing is lost.";
    root.appendChild(body);

    const err = document.createElement("div");
    err.id = "clique-update-err";
    err.style.cssText = "font:12px/1.4 'Segoe UI',system-ui,sans-serif;min-height:0;margin:0 0 8px;color:#f85149;";
    root.appendChild(err);

    const row = document.createElement("div");
    row.id = "clique-update-actions";
    row.style.cssText = "display:flex;gap:8px;justify-content:flex-end;align-items:center;";

    const later = document.createElement("button");
    later.id = "clique-update-later";
    later.type = "button";
    later.textContent = "Later";
    later.style.cssText = "font:600 13px/1.3 'Segoe UI',system-ui,sans-serif;padding:8px 12px;margin:0;border-radius:6px;cursor:pointer;background:transparent;color:#cccccc;border:1px solid #555555;";
    later.addEventListener("click", function () {
      root.remove();
    });
    row.appendChild(later);

    const restart = document.createElement("button");
    restart.id = "clique-update-restart";
    restart.type = "button";
    restart.textContent = "Restart now";
    restart.style.cssText = "font:600 13px/1.3 'Segoe UI',system-ui,sans-serif;padding:8px 12px;margin:0;border-radius:6px;cursor:pointer;background:#0078d4;color:#ffffff;border:1px solid #0078d4;";
    restart.addEventListener("click", async function () {
      later.disabled = true;
      restart.disabled = true;
      restart.textContent = packaged ? "Restarting..." : "Downloading...";
      err.textContent = "";
      try {
        const msg = await window.cliqueRestart();
        if (msg) window.__cliqueUpdateFailed(msg);
      } catch (e) {
        window.__cliqueUpdateFailed(e && e.message ? e.message : String(e));
      }
    });
    row.appendChild(restart);

    root.appendChild(row);
    (document.body || document.documentElement).appendChild(root);
    requestAnimationFrame(function () {
      root.classList.add("clique-update-in");
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

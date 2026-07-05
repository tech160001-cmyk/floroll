(function () {
  var modal = document.getElementById("modal");
  var content = document.getElementById("modal-content");
  var titleEl = document.getElementById("modal-title");
  if (!modal || !content) {
    return;
  }

  function open(title) {
    if (titleEl && title) {
      titleEl.textContent = title;
    }
    modal.hidden = false;
    requestAnimationFrame(function () {
      modal.classList.add("is-open");
    });
    modal.setAttribute("aria-hidden", "false");
    document.body.classList.add("modal-open");
    var closeBtn = modal.querySelector(".modal-close");
    if (closeBtn) {
      closeBtn.focus();
    }
  }

  function close() {
    modal.classList.remove("is-open");
    modal.setAttribute("aria-hidden", "true");
    document.body.classList.remove("modal-open");
    window.setTimeout(function () {
      if (!modal.classList.contains("is-open")) {
        modal.hidden = true;
        content.innerHTML = "";
        if (titleEl) {
          titleEl.textContent = "";
        }
      }
    }, 250);
  }

  window.Modal = { open: open, close: close };

  document.addEventListener("click", function (event) {
    if (event.target.closest("[data-modal-close]")) {
      event.preventDefault();
      close();
    }
  });

  document.addEventListener("keydown", function (event) {
    if (event.key === "Escape" && modal.classList.contains("is-open")) {
      close();
    }
  });

  document.body.addEventListener("htmx:afterSwap", function (event) {
    if (event.detail.target !== content) {
      return;
    }

    var trigger = event.detail.requestConfig && event.detail.requestConfig.elt;
    if (trigger && trigger.closest("#modal-content")) {
      return;
    }

    if (trigger && trigger.classList.contains("modal-trigger")) {
      open(trigger.getAttribute("data-modal-title") || "");
    }
  });
})();

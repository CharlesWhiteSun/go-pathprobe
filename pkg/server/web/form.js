'use strict';

// ── form.js — UI initialisation, form dynamics, custom select ─────────────
// Self-contained module responsible for:
//   • val() / checked() — typed form-field accessors
//   • getRunningHTML()  — run-button animation markup
//   • getModeFor()      — read checked sub-mode radio for a target
//   • applyModePanels() / updatePortGroup() — form layout dynamics
//   • measurePanelHeight() — off-screen height measurement
//   • onTargetChange()  — target-switch handler (panels + port fill + placeholder)
//   • initCustomSelect() — .cs-wrap keyboard/accessible dropdown widget
//   • initAdvancedOpts() — animated open/close for the Advanced Options panel
//   • initEnterKey()    — document-level Enter-key → runDiag() delegation
//   • init()            — wraps all form-related DOMContentLoaded setup
//
// Dependencies (runtime-resolved, no hard import):
//   • window.PathProbe.Config — TARGET_PORTS, TARGET_MODE_PANELS,
//     WEB_MODES_WITH_PORTS (declared in config.js)
//   • window.PathProbe.Locale.t — translation function (declared in locale.js)
//
// This module must be loaded AFTER config.js, locale.js and theme.js
// but BEFORE app.js.
(() => {
  // ── Config aliases (runtime-resolved) ─────────────────────────────────
  function _cfg() {
    return (window.PathProbe && window.PathProbe.Config) || {};
  }

  function _targetPorts()           { return _cfg().TARGET_PORTS           || {}; }
  function _targetModePanels()      { return _cfg().TARGET_MODE_PANELS      || {}; }
  function _webModesWithPorts()     { return _cfg().WEB_MODES_WITH_PORTS    || []; }
  function _webModesHideHost()      { return _cfg().WEB_MODES_HIDE_HOST     || []; }

  // ── Locale (runtime-resolved) ─────────────────────────────────────────
  /** Return the translation for key in the current locale, falling back to key. */
  function _t(key) {
    return (window.PathProbe && window.PathProbe.Locale && window.PathProbe.Locale.t)
      ? window.PathProbe.Locale.t(key)
      : key;
  }

  // ── State ──────────────────────────────────────────────────────────────
  // Track the first onTargetChange() call so no enter-animation plays on cold
  // page load (the form's initial state is already fully visible in the HTML).
  let _initTargetDone = false;

  /**
   * Cleanup function for any in-flight sequential panel transition.
   * Calling it cancels the pending animationend listeners and immediately
   * hides all departing panels so a rapid target switch always wins.
   */
  let _pendingReveal = null;

  // ── Form field accessors ───────────────────────────────────────────────

  /** Read and trim the string value of a form element by id. */
  function val(id) {
    const el = document.getElementById(id);
    return el ? el.value.trim() : '';
  }

  /** Return the checked state of a checkbox by id. */
  function checked(id) {
    const el = document.getElementById(id);
    return el ? el.checked : false;
  }

  // ── Run-button animation ───────────────────────────────────────────────
  /**
   * Return the innerHTML to inject into #run-btn while a diagnostic is running.
   * Uses the dots animation (three bouncing dots).
   */
  function getRunningHTML() {
    return '<span class="anim-dots"><span></span><span></span><span></span></span>';
  }

  // ── Form dynamics ──────────────────────────────────────────────────────

  /** Read the currently-checked sub-mode radio for a target. */
  function getModeFor(target) {
    const el = document.querySelector(`input[name="${target}-mode"]:checked`);
    return el ? el.value : '';
  }

  /** Show/hide mode-specific sub-panels for a target based on the checked mode radio. */
  function applyModePanels(target) {
    const mode   = getModeFor(target);
    const panels = (_targetModePanels()[target] || {});
    Object.entries(panels).forEach(([id, visibleModes]) => {
      const panel = document.getElementById(id);
      if (panel) panel.hidden = !visibleModes.includes(mode);
    });
  }

  /**
   * Show or hide the #port-group column and its text-input variant based on the
   * current target and mode.  Driven by WEB_MODES_WITH_PORTS so adding a new
   * web mode that needs ports only requires updating that constant.
   *
   * Rules:
   *   - web + mode in WEB_MODES_WITH_PORTS → show port-group + text input
   *   - web + other modes                  → hide port-group entirely
   *   - non-web targets                    → show port-group + text input
   */
  function updatePortGroup(target, mode) {
    const group   = document.getElementById('port-group');
    const textGrp = document.getElementById('ports-text-group');

    const needsPorts = ((target === 'web') && _webModesWithPorts().includes(mode)) || (target !== 'web');

    if (group)   group.hidden   = !needsPorts;
    if (textGrp) textGrp.hidden = !needsPorts;
  }

  /**
   * Show or hide the Target Host field (#host-group) and the DNS Domains field
   * (#dns-domains-group) based on the current target and mode.
   *
   * Rules:
   *   - web + mode in WEB_MODES_HIDE_HOST → hide #host-group, show #dns-domains-group
   *   - all other combinations            → show #host-group, hide #dns-domains-group
   */
  function updateHostGroup(target, mode) {
    const hostGrp = document.getElementById('host-group');
    const dnsGrp  = document.getElementById('dns-domains-group');
    const hideHost = (target === 'web') && _webModesHideHost().includes(mode);
    if (hostGrp) hostGrp.hidden = hideHost;
    if (dnsGrp)  dnsGrp.hidden  = !hideHost;
  }

  // ── Panel height measurement ───────────────────────────────────────────
  /**
   * Measure the layout height a panel element would occupy inside the stage
   * when visible.  The value matches what panel-stage.scrollHeight returns with
   * that panel present: offsetHeight (content + padding + border) plus any CSS
   * margins.  Using scrollHeight instead would be off by the border widths,
   * causing a visible snap when height:auto is restored after the transition.
   * Uses a detached clone so the live DOM is never modified.
   * @param  {HTMLElement} el         The panel to measure.
   * @param  {number}      stageWidth The layout width to simulate (matches .panel-stage).
   * @returns {number} Height in CSS pixels.
   */
  function measurePanelHeight(el, stageWidth) {
    const clone = el.cloneNode(true);
    clone.hidden = false;
    clone.style.cssText = [
      'position: absolute',
      'top: -9999px',
      'left: 0',
      'width: ' + (stageWidth || 300) + 'px',
      'visibility: hidden',
      'pointer-events: none',
    ].join('; ');
    document.body.appendChild(clone);
    // offsetHeight includes content + padding + border (unlike scrollHeight which
    // excludes border), so it exactly matches the child's contribution to the
    // parent container's scrollHeight.  Add CSS margins on top to get the total
    // space the element occupies inside an overflow:hidden stage.
    const cs           = getComputedStyle(clone);
    const marginTop    = parseFloat(cs.marginTop)    || 0;
    const marginBottom = parseFloat(cs.marginBottom) || 0;
    const h            = clone.offsetHeight + marginTop + marginBottom;
    document.body.removeChild(clone);
    return h;
  }

  // ── Target change handler ──────────────────────────────────────────────
  function onTargetChange() {
    const target  = val('target');
    const animate = _initTargetDone;
    _initTargetDone = true;

    // Cancel any previous in-flight transition before starting a new one.
    if (_pendingReveal) {
      _pendingReveal();
      _pendingReveal = null;
    }

    const incoming = document.getElementById('fields-' + target);
    if (!incoming) return;

    // Panels marked data-panel-empty carry no form content (e.g. imap, pop).
    // All departing panels are still hidden, but the incoming panel is never
    // revealed — so the user never sees an empty bordered box.
    const isEmptyPanel = incoming.dataset.panelEmpty === 'true';

    const stage = document.getElementById('panel-stage');

    // Collect all currently-visible panels that need to depart.
    const departing = Array.from(document.querySelectorAll('.target-fields'))
      .filter(fs => fs !== incoming && !fs.hidden);

    /**
     * Reveal the incoming panel with an optional enter animation.
     * Called only after all departing panels have finished their exit, so the
     * two animations are strictly sequential — no layout overlap.
     * The stage height is already at the incoming panel's measured height
     * (set during the departure phase), so no visual jump occurs on reveal.
     */
    function revealIncoming() {
      _pendingReveal = null;
      incoming.classList.remove('panel-leaving');
      if (isEmptyPanel) {
        // Empty panel: keep the fieldset hidden; collapse the stage back to
        // auto height (naturally 0 since no visible children remain).
        if (stage) stage.style.height = '';
        return;
      }
      incoming.hidden = false;
      if (animate) {
        incoming.classList.remove('panel-entering');
        void incoming.offsetWidth; // force reflow so animation restarts cleanly
        incoming.classList.add('panel-entering');
        // Restore auto height once the entrance animation is done so the stage
        // can grow/shrink naturally afterwards (e.g. sub-mode panel toggles).
        incoming.addEventListener('animationend', () => {
          if (stage) stage.style.height = '';
        }, { once: true });
      } else if (stage) {
        stage.style.height = '';
      }
    }

    if (animate && departing.length > 0) {
      // ── Sequential + height-animated transition ──────────────────────────
      // 1. Lock the stage at its current pixel height (enables CSS transition).
      // 2. Measure the incoming panel height via a detached clone.
      // 3. Set stage to the incoming height — both the height transition and the
      //    departure animation run in parallel over the same --panel-anim-dur.
      // 4. After all departing panels finish, reveal the incoming panel.
      if (stage) {
        const currentH  = stage.scrollHeight;
        // Empty panels target height 0 so the stage collapses smoothly.
        const incomingH = isEmptyPanel ? 0 : measurePanelHeight(incoming, stage.offsetWidth);
        stage.style.height = currentH + 'px';   // lock to pixels so transition works
        void stage.offsetHeight;                 // force reflow
        stage.style.height = incomingH + 'px';  // trigger height CSS transition
      }

      // Keep the incoming panel hidden while the outgoing content departs.
      incoming.hidden = true;

      let pending = departing.length;
      const listeners = [];

      departing.forEach(fs => {
        fs.classList.remove('panel-entering');
        fs.classList.add('panel-leaving');

        const handler = () => {
          fs.hidden = true;
          fs.classList.remove('panel-leaving');
          pending -= 1;
          if (pending === 0) revealIncoming();
        };

        fs.addEventListener('animationend', handler, { once: true });
        listeners.push({ fs, handler });
      });

      // Store cleanup so a rapid switch can cancel this flight.
      _pendingReveal = () => {
        if (stage) stage.style.height = '';
        listeners.forEach(({ fs, handler }) => {
          fs.removeEventListener('animationend', handler);
          fs.hidden = true;
          fs.classList.remove('panel-leaving', 'panel-entering');
        });
      };
    } else if (animate && !isEmptyPanel && stage) {
      // ── Grow from empty stage ─────────────────────────────────────────────
      // The previous target was an empty panel (never revealed, so departing=[]).
      // The stage is at height 0; animate it up to the incoming panel height
      // while the incoming panel fades in — producing the same smooth effect as
      // the symmetric "collapse to empty" transition in the opposite direction.
      document.querySelectorAll('.target-fields').forEach(fs => {
        if (fs !== incoming) {
          fs.hidden = true;
          fs.classList.remove('panel-entering', 'panel-leaving');
        }
      });
      const incomingH = measurePanelHeight(incoming, stage.offsetWidth);
      stage.style.height = '0px';       // lock at 0 so CSS transition has a start
      void stage.offsetHeight;           // force reflow
      stage.style.height = incomingH + 'px'; // trigger height CSS transition
      revealIncoming();
    } else {
      // Cold load or no visible departing panel and no animation: instant switch.
      document.querySelectorAll('.target-fields').forEach(fs => {
        if (fs !== incoming) {
          fs.hidden = true;
          fs.classList.remove('panel-entering', 'panel-leaving');
        }
      });
      if (stage) stage.style.height = '';
      revealIncoming();
    }

    // Non-animation updates — apply immediately, do not wait for transition.
    const portEl = document.getElementById('ports');
    if (portEl && portEl.dataset.userEdited !== 'true') {
      portEl.value = (_targetPorts()[target] || []).join(', ');
    }
    const hostEl = document.getElementById('host');
    if (hostEl) {
      hostEl.placeholder = _t('ph-host');
    }
    applyModePanels(target);
    updatePortGroup(target, getModeFor(target));
    updateHostGroup(target, getModeFor(target));
  }

  // ── Enter-key submit ───────────────────────────────────────────────────
  /**
   * Attach a document-level keydown listener so pressing Enter inside any
   * text or number input triggers the diagnostic run (window.runDiag).
   *
   * Guards:
   *   – e.isComposing  → skip mid-IME-composition input (CJK, etc.)
   *   – run-btn is disabled → skip when a diagnostic is already running
   *   – non-INPUT / non-TEXTAREA target → skip (checkboxes, radios, buttons …)
   *
   * Event delegation on document means all current and future input fields
   * are covered without per-element wiring.
   */
  function initEnterKey() {
    document.addEventListener('keydown', function onEnterKey(e) {
      if (e.key !== 'Enter') return;
      if (e.isComposing) return;                         // IME: not yet committed
      const tag = e.target && e.target.tagName;
      if (tag !== 'INPUT' && tag !== 'TEXTAREA') return;
      const btn = document.getElementById('run-btn');
      if (btn && btn.disabled) return;                   // diagnostic already running
      e.preventDefault();                                // prevent browser form submit
      if (window.runDiag) window.runDiag();
    });
  }

  // ── Advanced Options animated expand/collapse ──────────────────────────
  /**
   * Wire up animated open/close for the Advanced Options <details> element.
   * Intercepts summary clicks and drives a height transition on .adv-body
   * (mirroring the panel-stage mechanism) together with a fade+slide animation
   * on .adv-inner, reusing the panel-appear / panel-leave keyframes and the
   * --panel-anim-dur token so vivid / off modes apply automatically.
   */
  function initAdvancedOpts() {
    const details = document.getElementById('advanced-opts');
    if (!details) return;
    const summary = details.querySelector(':scope > summary');
    const body    = details.querySelector('.adv-body');
    if (!summary || !body) return;

    summary.addEventListener('click', e => {
      e.preventDefault();

      if (details.open) {
        // ── Collapse: animate from current height to 0, then toggle off ────
        details.classList.remove('adv-is-open'); // start chevron rotation immediately
        const currentH = body.scrollHeight;
        body.style.height = currentH + 'px';
        void body.offsetHeight;                    // flush reflow before transition
        body.classList.remove('adv-entering');
        body.classList.add('adv-leaving');
        body.style.height = '0px';

        body.addEventListener('transitionend', () => {
          details.open = false;
          body.classList.remove('adv-leaving');
          body.style.height = '';
        }, { once: true });
      } else {
        // ── Expand: open, measure, animate from 0 to full height ───────────
        details.open = true;
        details.classList.add('adv-is-open');    // start chevron rotation immediately
        const targetH = body.scrollHeight;
        body.style.height = '0px';
        void body.offsetHeight;                    // flush reflow before transition
        body.classList.remove('adv-leaving');
        body.classList.add('adv-entering');
        body.style.height = targetH + 'px';

        body.addEventListener('transitionend', () => {
          body.classList.remove('adv-entering');
          body.style.height = '';
        }, { once: true });
      }
    });
  }

  // ── Custom select component (cs-*) ─────────────────────────────────────
  /**
   * Initialise one .cs-wrap widget.
   * Syncs the hidden native <select> so val() continues to work without
   * modifications elsewhere.  Full keyboard support (Enter/Space to open,
   * ↑↓ to navigate, Escape/Tab to close).
   */
  function initCustomSelect(wrap) {
    const trigger = wrap.querySelector('.cs-trigger');
    const label   = wrap.querySelector('.cs-label');
    const list    = wrap.querySelector('.cs-list');
    const select  = wrap.querySelector('select');
    const items   = Array.from(wrap.querySelectorAll('.cs-item'));
    if (!trigger || !list || !items.length) return;

    // Mark the widget as having a selection as soon as it is initialised.
    // This applies the persistent primary-border indicator (like radio :checked)
    // regardless of keyboard focus state.
    wrap.classList.add('has-selection');

    /**
     * Close the popup.
     * @param {boolean} [restoreFocus=true] When true, return keyboard focus to
     *   the trigger (normal close via key or item select).  Pass false when
     *   closing because the user clicked OUTSIDE the widget so we do not steal
     *   focus away from whichever element they just clicked.
     */
    function close(restoreFocus = true) {
      wrap.classList.remove('open');
      trigger.setAttribute('aria-expanded', 'false');
      if (restoreFocus) trigger.focus();
    }

    function open() {
      wrap.classList.add('open');
      trigger.setAttribute('aria-expanded', 'true');
      const sel = list.querySelector('[aria-selected="true"]') || items[0];
      if (sel) sel.focus();
    }

    function selectItem(item) {
      items.forEach(it => it.removeAttribute('aria-selected'));
      item.setAttribute('aria-selected', 'true');
      // Ensure persistent selection indicator remains active after choice.
      wrap.classList.add('has-selection');
      // Sync the visible label (keep data-i18n so applyLocale() can re-translate).
      if (label) {
        label.textContent  = item.textContent;
        label.dataset.i18n = item.dataset.i18n || '';
      }
      // Sync hidden native select so val('target') always reads the correct value.
      if (select) select.value = item.dataset.value || '';
      // Trigger dependent logic directly (no change event on a hidden element).
      if (wrap.id === 'target-wrap') onTargetChange();
      close();
    }

    trigger.addEventListener('click', () => {
      wrap.classList.contains('open') ? close() : open();
    });

    trigger.addEventListener('keydown', e => {
      if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); open(); }
      if (e.key === 'Escape') close();
      if (e.key === 'ArrowDown') { e.preventDefault(); open(); }
    });

    items.forEach((item, idx) => {
      item.setAttribute('tabindex', '-1');
      item.addEventListener('click', () => selectItem(item));
      item.addEventListener('keydown', e => {
        if (e.key === 'Enter' || e.key === ' ')  { e.preventDefault(); selectItem(item); }
        if (e.key === 'ArrowDown') { e.preventDefault(); (items[idx + 1] || items[idx]).focus(); }
        if (e.key === 'ArrowUp')   { e.preventDefault(); (items[idx - 1] || items[idx]).focus(); }
        if (e.key === 'Escape' || e.key === 'Tab') { e.preventDefault(); close(); }
      });
    });

    // Close when focus moves outside the widget via an outside click.
    // restoreFocus=false: do NOT steal focus from the element the user just clicked.
    document.addEventListener('click', e => {
      if (!wrap.contains(e.target)) close(false);
    }, true);
  }

  // ── Form initialisation ────────────────────────────────────────────────
  /**
   * Initialise all form-related UI setup.  Intended to be called from the
   * app's DOMContentLoaded handler so all DOM elements are accessible.
   */
  function init() {
    // Disable browser spell-check, auto-correct and auto-capitalise on every
    // text input.  All fields contain technical identifiers (hostnames, URLs,
    // ports, credentials) where the browser's natural-language heuristics
    // produce misleading red-underline noise rather than useful feedback.
    // Centralising this here means newly added inputs are covered automatically.
    document.querySelectorAll('input[type="text"]').forEach(el => {
      el.spellcheck = false;
      el.setAttribute('autocorrect', 'off');
      el.setAttribute('autocapitalize', 'none');
    });

    // Track whether the user has manually edited the auto-filled fields.
    ['host', 'ports'].forEach(id => {
      const el = document.getElementById(id);
      if (el) el.addEventListener('input', () => { el.dataset.userEdited = 'true'; });
    });

    onTargetChange(); // populate defaults for initial selection

    // Hook up all sub-mode radio buttons generically.
    document.querySelectorAll('input[type="radio"][name$="-mode"]').forEach(radio => {
      radio.addEventListener('change', () => {
        const target = radio.name.replace(/-mode$/, '');
        applyModePanels(target);
        updatePortGroup(target, getModeFor(target));
        updateHostGroup(target, getModeFor(target));
        // Auto-fill ports when switching to a port-needing web mode
        // (mirrors the auto-fill that onTargetChange() does on target switch).
        if (target === 'web') {
          const portEl = document.getElementById('ports');
          if (portEl && portEl.dataset.userEdited !== 'true') {
            const mode = getModeFor(target);
            if (_webModesWithPorts().includes(mode)) {
              portEl.value = (_targetPorts()[target] || []).join(', ');
            }
          }
        }
      });
    });

    // Initialise all custom-select widgets on the page.
    document.querySelectorAll('.cs-wrap').forEach(wrap => initCustomSelect(wrap));

    // Wire up animated open/close for the advanced options panel.
    initAdvancedOpts();

    // Wire up Enter key to submit the diagnostic from any input field.
    initEnterKey();
  }

  // ── Namespace registration ─────────────────────────────────────────────
  const PathProbe = window.PathProbe || {};
  PathProbe.Form = { val, checked, getModeFor, getRunningHTML, onTargetChange, init };
  window.PathProbe = PathProbe;
})(); // end form IIFE

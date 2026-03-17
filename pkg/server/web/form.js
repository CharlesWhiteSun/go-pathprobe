'use strict';
// ── form.js — UI initialisation, form dynamics, custom select (PathProbe.Form)
// Depends on: config.js (PathProbe.Config), locale.js (PathProbe.Locale)
const PathProbe = window.PathProbe || {};
window.PathProbe = PathProbe;

// Module-level state for panel transition sequencing.
let _initTargetDone = false;
let _pendingReveal  = null;

// ── DOM helpers ───────────────────────────────────────────────────────────

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

/** Return the innerHTML to inject into #run-btn while a diagnostic is running. */
function getRunningHTML() {
  return '<span class="anim-dots"><span></span><span></span><span></span></span>';
}

// ── Form dynamics ─────────────────────────────────────────────────────────

function getModeFor(target) {
  const el = document.querySelector(`input[name="${target}-mode"]:checked`);
  return el ? el.value : '';
}

function applyModePanels(target) {
  const mode   = getModeFor(target);
  const panels = (PathProbe.Config.TARGET_MODE_PANELS[target] || {});
  Object.entries(panels).forEach(([id, visibleModes]) => {
    const panel = document.getElementById(id);
    if (panel) panel.hidden = !visibleModes.includes(mode);
  });
}

/**
 * Show or hide the #port-group column and its text-input variant based on the
 * current target and mode.
 */
function updatePortGroup(target, mode) {
  const group   = document.getElementById('port-group');
  const textGrp = document.getElementById('ports-text-group');
  const { WEB_MODES_WITH_PORTS } = PathProbe.Config;
  const needsPorts = ((target === 'web') && WEB_MODES_WITH_PORTS.includes(mode)) || (target !== 'web');
  if (group)   group.hidden   = !needsPorts;
  if (textGrp) textGrp.hidden = !needsPorts;
}

/**
 * Measure the layout height a panel element would occupy inside the stage
 * when visible, using a detached clone to avoid touching the live DOM.
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
  const cs           = getComputedStyle(clone);
  const marginTop    = parseFloat(cs.marginTop)    || 0;
  const marginBottom = parseFloat(cs.marginBottom) || 0;
  const h            = clone.offsetHeight + marginTop + marginBottom;
  document.body.removeChild(clone);
  return h;
}

function onTargetChange() {
  const { TARGET_PORTS, TARGET_PLACEHOLDER_KEYS, WEB_MODES_WITH_PORTS } = PathProbe.Config;
  const t       = (key) => PathProbe.Locale.t(key);
  const target  = val('target');
  const animate = _initTargetDone;
  _initTargetDone = true;

  if (_pendingReveal) {
    _pendingReveal();
    _pendingReveal = null;
  }

  const incoming    = document.getElementById('fields-' + target);
  if (!incoming) return;
  const isEmptyPanel = incoming.dataset.panelEmpty === 'true';
  const stage        = document.getElementById('panel-stage');
  const departing    = Array.from(document.querySelectorAll('.target-fields'))
    .filter(fs => fs !== incoming && !fs.hidden);

  function revealIncoming() {
    _pendingReveal = null;
    incoming.classList.remove('panel-leaving');
    if (isEmptyPanel) {
      if (stage) stage.style.height = '';
      return;
    }
    incoming.hidden = false;
    if (animate) {
      incoming.classList.remove('panel-entering');
      void incoming.offsetWidth;
      incoming.classList.add('panel-entering');
      incoming.addEventListener('animationend', () => {
        if (stage) stage.style.height = '';
      }, { once: true });
    } else if (stage) {
      stage.style.height = '';
    }
  }

  if (animate && departing.length > 0) {
    if (stage) {
      const currentH  = stage.scrollHeight;
      const incomingH = isEmptyPanel ? 0 : measurePanelHeight(incoming, stage.offsetWidth);
      stage.style.height = currentH + 'px';
      void stage.offsetHeight;
      stage.style.height = incomingH + 'px';
    }
    incoming.hidden = true;
    let pending  = departing.length;
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
    _pendingReveal = () => {
      if (stage) stage.style.height = '';
      listeners.forEach(({ fs, handler }) => {
        fs.removeEventListener('animationend', handler);
        fs.hidden = true;
        fs.classList.remove('panel-leaving', 'panel-entering');
      });
    };
  } else if (animate && !isEmptyPanel && stage) {
    document.querySelectorAll('.target-fields').forEach(fs => {
      if (fs !== incoming) {
        fs.hidden = true;
        fs.classList.remove('panel-entering', 'panel-leaving');
      }
    });
    const incomingH = measurePanelHeight(incoming, stage.offsetWidth);
    stage.style.height = '0px';
    void stage.offsetHeight;
    stage.style.height = incomingH + 'px';
    revealIncoming();
  } else {
    document.querySelectorAll('.target-fields').forEach(fs => {
      if (fs !== incoming) {
        fs.hidden = true;
        fs.classList.remove('panel-entering', 'panel-leaving');
      }
    });
    if (stage) stage.style.height = '';
    revealIncoming();
  }

  // Non-animation updates — apply immediately.
  const portEl = document.getElementById('ports');
  if (portEl && portEl.dataset.userEdited !== 'true') {
    portEl.value = (TARGET_PORTS[target] || []).join(', ');
  }
  const hostEl = document.getElementById('host');
  if (hostEl) {
    hostEl.placeholder = t(TARGET_PLACEHOLDER_KEYS[target] || 'ph-host-default');
  }
  applyModePanels(target);
  updatePortGroup(target, getModeFor(target));
}

// ── Advanced-options animated expand/collapse ─────────────────────────────

function initAdvancedOpts() {
  const details = document.getElementById('advanced-opts');
  if (!details) return;
  const summary = details.querySelector(':scope > summary');
  const body    = details.querySelector('.adv-body');
  if (!summary || !body) return;

  summary.addEventListener('click', e => {
    e.preventDefault();
    if (details.open) {
      details.classList.remove('adv-is-open');
      const currentH = body.scrollHeight;
      body.style.height = currentH + 'px';
      void body.offsetHeight;
      body.classList.remove('adv-entering');
      body.classList.add('adv-leaving');
      body.style.height = '0px';
      body.addEventListener('transitionend', () => {
        details.open = false;
        body.classList.remove('adv-leaving');
        body.style.height = '';
      }, { once: true });
    } else {
      details.open = true;
      details.classList.add('adv-is-open');
      const targetH = body.scrollHeight;
      body.style.height = '0px';
      void body.offsetHeight;
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

// ── Custom select component (cs-*) ────────────────────────────────────────

function initCustomSelect(wrap) {
  const trigger = wrap.querySelector('.cs-trigger');
  const label   = wrap.querySelector('.cs-label');
  const list    = wrap.querySelector('.cs-list');
  const select  = wrap.querySelector('select');
  const items   = Array.from(wrap.querySelectorAll('.cs-item'));
  if (!trigger || !list || !items.length) return;

  wrap.classList.add('has-selection');

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
    wrap.classList.add('has-selection');
    if (label) {
      label.textContent  = item.textContent;
      label.dataset.i18n = item.dataset.i18n || '';
    }
    if (select) select.value = item.dataset.value || '';
    if (wrap.id === 'target-wrap') onTargetChange();
    close();
  }

  trigger.addEventListener('click', () => {
    wrap.classList.contains('open') ? close() : open();
  });

  trigger.addEventListener('keydown', e => {
    if (e.key === 'Enter' || e.key === ' ') { e.preventDefault(); open(); }
    if (e.key === 'Escape')    close();
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

  document.addEventListener('click', e => {
    if (!wrap.contains(e.target)) close(false);
  }, true);
}

// ── Module initialisation (called from app.js DOMContentLoaded) ───────────

function init() {
  const { TARGET_PORTS, WEB_MODES_WITH_PORTS } = PathProbe.Config;

  // Disable spell-check / auto-correct on all text inputs.
  document.querySelectorAll('input[type="text"]').forEach(el => {
    el.spellcheck = false;
    el.setAttribute('autocorrect', 'off');
    el.setAttribute('autocapitalize', 'none');
  });

  // Track user edits so auto-fill is not overridden after manual changes.
  ['host', 'ports'].forEach(id => {
    const el = document.getElementById(id);
    if (el) el.addEventListener('input', () => { el.dataset.userEdited = 'true'; });
  });

  onTargetChange();

  // Hook up all sub-mode radio buttons generically.
  document.querySelectorAll('input[type="radio"][name$="-mode"]').forEach(radio => {
    radio.addEventListener('change', () => {
      const target = radio.name.replace(/-mode$/, '');
      applyModePanels(target);
      updatePortGroup(target, getModeFor(target));
      if (target === 'web') {
        const portEl = document.getElementById('ports');
        if (portEl && portEl.dataset.userEdited !== 'true') {
          const mode = getModeFor(target);
          if (WEB_MODES_WITH_PORTS.includes(mode)) {
            portEl.value = (TARGET_PORTS[target] || []).join(', ');
          }
        }
      }
    });
  });

  document.querySelectorAll('.cs-wrap').forEach(wrap => initCustomSelect(wrap));
  initAdvancedOpts();
}

// ── Public API ────────────────────────────────────────────────────────────
PathProbe.Form = {
  val,
  checked,
  getModeFor,
  getRunningHTML,
  onTargetChange,
  init,
};

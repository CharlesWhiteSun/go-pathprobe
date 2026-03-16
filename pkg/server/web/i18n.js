'use strict';
/**
 * i18n.js — PathProbe UI translation dictionary.
 *
 * To add a new locale, append an entry to LOCALES keyed by the BCP-47 tag.
 * Every key missing from a non-English locale falls back to the 'en' entry
 * transparently via the t() helper in app.js.
 */
window.LOCALES = {
  en: {
    /* ── Form ─────────────────────────────────────────────────────────── */
    'run-diagnostic':        'Diagnostic',
    'label-target':          'Target',
    'label-host':            'Target Host',
    'label-ports':           'Ports',
    'label-ports-hint':      '(comma-separated)',

    /* ── Web / DNS fieldset ───────────────────────────────────────────── */
    'legend-web':            'Web / DNS Options',
    'label-web-mode':        'Detection Mode',
    'web-mode-public-ip':    'Public IP Detection',
    'web-mode-dns':          'DNS Comparison',
    'web-mode-http':         'HTTP / HTTPS Probe',
    'web-mode-port':         'Port Connectivity',
    'web-mode-traceroute':   'Route Trace',
    'label-dns-domains':     'DNS Domains',
    'label-dns-domains-hint':'(comma-separated)',
    'label-record-types':    'Record Types',
    'dns-type-A':            'IPv4',
    'dns-type-AAAA':         'IPv6',
    'dns-type-MX':           'MX',
    'ph-dns-domains':        'e.g. google.com',
    'label-http-url':        'HTTP URL',
    'label-http-url-hint':   '(optional probe)',
    'label-max-hops':        'Max Hops',

    /* ── SMTP fieldset ────────────────────────────────────────────────── */
    'legend-smtp':           'SMTP Options',
    'label-smtp-mode':       'Detection Mode',
    'smtp-mode-handshake':   'Banner & EHLO (no auth)',
    'smtp-mode-auth':        'Authentication test',
    'smtp-mode-send':        'Full send simulation',
    'label-smtp-domain':     'EHLO Domain',
    'label-smtp-user':       'Username',
    'label-smtp-pass':       'Password',
    'label-smtp-from':       'MAIL FROM',
    'label-smtp-to':         'RCPT TO',
    'label-smtp-to-hint':    '(comma-separated)',
    'label-smtp-starttls':   'STARTTLS',
    'label-smtp-ssl':        'Implicit TLS (SMTPS)',
    'label-smtp-mx-all':     'Probe all MX records',

    /* ── FTP fieldset ─────────────────────────────────────────────────── */
    'legend-imap':           'IMAP Options',
    'legend-pop':            'POP3 Options',
    'legend-ftp':            'FTP / FTPS Options',
    'label-ftp-mode':        'Detection Mode',
    'ftp-mode-login':        'Connect & login',
    'ftp-mode-list':         'Login + directory listing',
    'label-ftp-user':        'Username',
    'label-ftp-user-hint':   '(blank = anonymous)',
    'label-ftp-pass':        'Password',
    'label-ftp-ssl':         'Implicit FTPS (port 990)',
    'label-ftp-auth-tls':    'Explicit FTPS (AUTH TLS)',

    /* ── SFTP fieldset ────────────────────────────────────────────────── */
    'legend-sftp':           'SFTP / SSH Options',
    'label-sftp-mode':       'Detection Mode',
    'sftp-mode-auth':        'SSH authentication',
    'sftp-mode-ls':          'Auth + list directory',
    'label-sftp-user':       'Username',
    'label-sftp-pass':       'Password',
    'label-sftp-ls':         'List remote default directory',

    /* ── Advanced ─────────────────────────────────────────────────────── */
    'adv-summary':           'Advanced Options',
    'label-mtr-count':       'MTR Count',
    'label-timeout':         'Timeout',
    'label-insecure':        'Skip TLS certificate verification',
    'label-geo-enabled':     'Enable Geo annotation & map',

    /* ── Run button ───────────────────────────────────────────────────── */
    'btn-run':               '▶',
    'btn-running':           '',


    /* ── Error messages (UI-friendly versions of backend errors) ─────── */
    'err-timeout':           'Diagnostic timed out — the operation exceeded the configured timeout. For Route Trace, try increasing Timeout in Advanced Options or reducing Max Hops.',
    'err-no-runner':         'No handler registered for this diagnostic target.',
    'err-unknown':           'An unknown error occurred. Please try again.',

    /* ── Results & history ────────────────────────────────────────────── */
    'results-title':         'Results',
    'history-title':         'History',
    'history-empty':         'No diagnostics run yet.',

    /* ── Summary keys ─────────────────────────────────────────────────── */
    'key-target':            'Target',
    'key-host':              'Host',
    'key-generated':         'Generated',
    'key-public-ip':         'Public IP',

    /* ── Port connectivity table ──────────────────────────────────────── */
    'section-ports':         'Port Connectivity',
    'th-port':               'Port',
    'th-sent':               'Sent',
    'th-recv':               'Recv',
    'th-loss':               'Loss%',
    'th-min-rtt':            'Min RTT',
    'th-avg-rtt':            'Avg RTT',
    'th-max-rtt':            'Max RTT',

    /* ── Protocol results table ───────────────────────────────────────── */
    'section-protos':        'Protocol Results',
    'th-protocol':           'Protocol',
    'th-host':               'Host',
    'th-status':             'Status',
    'th-summary':            'Summary',

    /* ── Route trace table ────────────────────────────────────────────── */
    'section-route':         'Route Trace',
    'th-ttl':                'TTL',
    'th-ip-host':            'IP / Host',
    'th-asn':                'ASN',
    'th-country':            'Country',

    /* ── Geo section ──────────────────────────────────────────────────── */
    'section-geo':           'Geo Information',
    'geo-public-ip':         'Public IP',
    'geo-target-host':       'Target Host',
    'geo-no-data':           'No data',
    'geo-kv-ip':             'IP',
    'geo-kv-city':           'City',
    'geo-kv-country':        'Country',
    'geo-kv-asn':            'ASN',

    /* ── Map popup labels ─────────────────────────────────────────────── */
    'map-public-ip':         'Public IP',
    'map-origin':            'Origin',
    'map-target':            'Target',
    'map-distance':          'Distance',    'map-tile-light':        'Light',
    'map-tile-osm':          'OSM',
    'map-tile-dark':         'Dark',
    /* ── Marker style label ─────────────────────────────────────────────── */
    'marker-style-diamond-pulse':    'Pulse',
    /* ── Marker colour scheme ────────────────────────────────────────────── */
    'marker-color-ocean':      'Ocean',
    /* ── Connector line styles ─────────────────────────────────────── */
    'connector-tick-xs':       '> xs',
    /* ── Target select options ────────────────────────────────────── */
    'opt-web':               'WEB \u2014 Public IP & DNS',
    'opt-smtp':              'SMTP \u2014 Mail server',
    'opt-imap':              'IMAP \u2014 IMAP server',
    'opt-pop':               'POP \u2014 POP3 server',
    'opt-ftp':               'FTP \u2014 FTP / FTPS',
    'opt-sftp':              'SFTP \u2014 SFTP / SSH',

    /* ── Language switcher button labels ─────────────────────────────── */
    'btn-lang-en':           'EN',
    'btn-lang-zh':           'TW',

    /* ── Host input placeholders ──────────────────────────────────────── */
    'ph-web':                'e.g. google.com',
    'ph-smtp':               'e.g. mail.example.com',
    'ph-imap':               'e.g. mail.example.com',
    'ph-pop':                'e.g. mail.example.com',
    'ph-ftp':                'e.g. ftp.example.com',
    'ph-sftp':               'e.g. sftp.example.com',
    'ph-host-default':       'hostname or IP',

    /* ── Theme switcher ─────────────────────────────────────────────── */
    'theme-default':          'Default',
    'theme-deep-blue':        'Deep Blue',
    'theme-light-green':      'Light Green',
    'theme-forest-green':     'Forest Green',
    'theme-dark':             'Dark',

    /* ── Footer ───────────────────────────────────────────────────── */
    'footer-copyright':       '© 2026 Charles. All Rights Reserved.',
  },

  'zh-TW': {
    /* ── Form ─────────────────────────────────────────────────────────── */
    'run-diagnostic':        '診斷',
    'label-target':          '目標類型',
    'label-host':            '目標主機',
    'label-ports':           '連接埠',
    'label-ports-hint':      '（逗號分隔）',

    /* ── Web / DNS fieldset ───────────────────────────────────────────── */
    'legend-web':            'Web / DNS 選項',
    'label-web-mode':        '偵測模式',
    'web-mode-public-ip':    '公開 IP 偵測',
    'web-mode-dns':          'DNS 跨解析器比對',
    'web-mode-http':         'HTTP / HTTPS 探測',
    'web-mode-port':         '連接埠連通性',
    'web-mode-traceroute':   '路由追蹤',
    'label-dns-domains':     'DNS 網域',
    'label-dns-domains-hint':'（逗號分隔）',
    'label-record-types':    '記錄類型',
    'dns-type-A':            'IPv4',
    'dns-type-AAAA':         'IPv6',
    'dns-type-MX':           'MX',
    'ph-dns-domains':        '例如 google.com',
    'label-http-url':        'HTTP URL',
    'label-http-url-hint':   '（選填探測）',
    'label-max-hops':        '最大躍點數',

    /* ── SMTP fieldset ────────────────────────────────────────────────── */
    'legend-smtp':           'SMTP 選項',
    'label-smtp-mode':       '偵測模式',
    'smtp-mode-handshake':   'Banner & EHLO（無驗證）',
    'smtp-mode-auth':        '身分驗證測試',
    'smtp-mode-send':        '完整傳送流程模擬',
    'label-smtp-domain':     'EHLO 網域',
    'label-smtp-user':       '使用者名稱',
    'label-smtp-pass':       '密碼',
    'label-smtp-from':       '寄件者（MAIL FROM）',
    'label-smtp-to':         '收件者（RCPT TO）',
    'label-smtp-to-hint':    '（逗號分隔）',
    'label-smtp-starttls':   'STARTTLS',
    'label-smtp-ssl':        '隱式 TLS（SMTPS）',
    'label-smtp-mx-all':     '探測所有 MX 記錄',

    /* ── FTP fieldset ─────────────────────────────────────────────────── */
    'legend-imap':           'IMAP 選項',
    'legend-pop':            'POP3 選項',
    'legend-ftp':            'FTP / FTPS 選項',
    'label-ftp-mode':        '偵測模式',
    'ftp-mode-login':        '連線並登入',
    'ftp-mode-list':         '登入 + 目錄列表',
    'label-ftp-user':        '使用者名稱',
    'label-ftp-user-hint':   '（留空 = 匿名）',
    'label-ftp-pass':        '密碼',
    'label-ftp-ssl':         '隱式 FTPS（連接埠 990）',
    'label-ftp-auth-tls':    '顯式 FTPS（AUTH TLS）',

    /* ── SFTP fieldset ────────────────────────────────────────────────── */
    'legend-sftp':           'SFTP / SSH 選項',
    'label-sftp-mode':       '偵測模式',
    'sftp-mode-auth':        'SSH 身分驗證',
    'sftp-mode-ls':          '驗證 + 列出目錄',
    'label-sftp-user':       '使用者名稱',
    'label-sftp-pass':       '密碼',
    'label-sftp-ls':         '列出遠端預設目錄',

    /* ── Advanced ─────────────────────────────────────────────────────── */
    'adv-summary':           '進階選項',
    'label-mtr-count':       'MTR 次數',
    'label-timeout':         '逾時時間',
    'label-insecure':        '略過 TLS 憑證驗證',
    'label-geo-enabled':     '啟用地理標註與地圖',

    /* ── Run button ───────────────────────────────────────────────────── */
    'btn-run':               '▶',
    'btn-running':           '',


    /* ── Error messages (UI-friendly versions of backend errors) ─────── */
    'err-timeout':           '診斷逾時 — 操作超過設定的等待時間。若使用路由追蹤，請嘗試在進階選項中增加逾時時間，或縮短最大躍點數。',
    'err-no-runner':         '找不到此診斷目標所對應的處理器。',
    'err-unknown':           '發生未知錯誤，請重試。',

    /* ── Results & history ────────────────────────────────────────────── */
    'results-title':         '結果',
    'history-title':         '記錄',
    'history-empty':         '尚無診斷記錄。',

    /* ── Summary keys ─────────────────────────────────────────────────── */
    'key-target':            '目標類型',
    'key-host':              '主機',
    'key-generated':         '產生時間',
    'key-public-ip':         '公開 IP',

    /* ── Port connectivity table ──────────────────────────────────────── */
    'section-ports':         '連接埠連通性',
    'th-port':               '連接埠',
    'th-sent':               '已送',
    'th-recv':               '已收',
    'th-loss':               '丟包率%',
    'th-min-rtt':            '最小 RTT',
    'th-avg-rtt':            '平均 RTT',
    'th-max-rtt':            '最大 RTT',

    /* ── Protocol results table ───────────────────────────────────────── */
    'section-protos':        '協定結果',
    'th-protocol':           '協定',
    'th-host':               '主機',
    'th-status':             '狀態',
    'th-summary':            '摘要',

    /* ── Route trace table ────────────────────────────────────────────── */
    'section-route':         '路由路徑',
    'th-ttl':                'TTL',
    'th-ip-host':            'IP / 主機',
    'th-asn':                'ASN',
    'th-country':            '國家',

    /* ── Geo section ──────────────────────────────────────────────────── */
    'section-geo':           '地理資訊',
    'geo-public-ip':         '公開 IP',
    'geo-target-host':       '目標主機',
    'geo-no-data':           '無資料',
    'geo-kv-ip':             'IP',
    'geo-kv-city':           '城市',
    'geo-kv-country':        '國家',
    'geo-kv-asn':            'ASN',

    /* ── Map popup labels ─────────────────────────────────────────────── */
    'map-public-ip':         '公開 IP',
    'map-origin':            '偵測起點',
    'map-target':            '偵測目標',
    'map-distance':          '連線距離',    'map-tile-light':        '淡色',
    'map-tile-osm':          '原始風格',
    'map-tile-dark':         '深色',
    /* ── 標記外觀標籤 ─────────────────────────────────────────── */
    'marker-style-diamond-pulse':    '脈衝',
    /* ── 標記配色方案 ────────────────────────────────────────── */
    'marker-color-ocean':      '海洋',
    /* ── 連接弧線風格 ─────────────────────────────────── */
    'connector-tick-xs':       '> xs',
    /* ── Target select options ────────────────────────────────────────── */
    'opt-web':               'WEB \u2014 公開 IP 與 DNS',
    'opt-smtp':              'SMTP \u2014 郵件伺服器',
    'opt-imap':              'IMAP \u2014 IMAP 伺服器',
    'opt-pop':               'POP \u2014 POP3 伺服器',
    'opt-ftp':               'FTP \u2014 FTP / FTPS',
    'opt-sftp':              'SFTP \u2014 SFTP / SSH',

    /* ── Language switcher button labels ─────────────────────────────── */
    'btn-lang-en':           '英文',
    'btn-lang-zh':           '繁中',

    /* ── Host input placeholders ──────────────────────────────────────── */
    'ph-web':                '例：google.com',
    'ph-smtp':               '例：mail.example.com',
    'ph-imap':               '例：mail.example.com',
    'ph-pop':                '例：mail.example.com',
    'ph-ftp':                '例：ftp.example.com',
    'ph-sftp':               '例：sftp.example.com',
    'ph-host-default':       '主機名稱或 IP',

    /* ── Theme switcher ─────────────────────────────────────────────── */
    'theme-default':          '預設',
    'theme-deep-blue':        '深藍',
    'theme-light-green':      '淡綠',
    'theme-forest-green':     '墨綠',
    'theme-dark':             '暗黑',

    /* ── Footer ───────────────────────────────────────────────────── */
    'footer-copyright':       '© 2026 Charles．保留所有權利。',
  },
};

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
    'run-diagnostic':        'Run Diagnostic',
    'label-target':          'Target',
    'label-host':            'Target Host',
    'label-ports':           'Ports',
    'label-ports-hint':      '(comma-separated)',

    /* ── Web / DNS fieldset ───────────────────────────────────────────── */
    'legend-web':            'Web / DNS Options',
    'label-dns-domains':     'DNS Domains',
    'label-dns-domains-hint':'(comma-separated)',
    'label-record-types':    'Record Types',
    'label-http-url':        'HTTP URL',
    'label-http-url-hint':   '(optional probe)',

    /* ── SMTP fieldset ────────────────────────────────────────────────── */
    'legend-smtp':           'SMTP Options',
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
    'legend-ftp':            'FTP / FTPS Options',
    'label-ftp-user':        'Username',
    'label-ftp-user-hint':   '(blank = anonymous)',
    'label-ftp-pass':        'Password',
    'label-ftp-ssl':         'Implicit FTPS (port 990)',
    'label-ftp-auth-tls':    'Explicit FTPS (AUTH TLS)',
    'label-ftp-list':        'Attempt PASV + LIST',

    /* ── SFTP fieldset ────────────────────────────────────────────────── */
    'legend-sftp':           'SFTP / SSH Options',
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
    'btn-run':               '\u25b6 Run Diagnostic',
    'btn-running':           'Running\u2026',

    /* ── Results & history ────────────────────────────────────────────── */
    'results-title':         'Results',
    'history-title':         'Diagnostic History',
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
    'map-target':            'Target',

    /* ── Target select options ────────────────────────────────────────── */
    'opt-web':               'web \u2014 Public IP & DNS',
    'opt-smtp':              'smtp \u2014 Mail server',
    'opt-imap':              'imap \u2014 IMAP server',
    'opt-pop':               'pop \u2014 POP3 server',
    'opt-ftp':               'ftp \u2014 FTP / FTPS',
    'opt-sftp':              'sftp \u2014 SFTP / SSH',

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
  },

  'zh-TW': {
    /* ── Form ─────────────────────────────────────────────────────────── */
    'run-diagnostic':        '執行診斷',
    'label-target':          '目標類型',
    'label-host':            '目標主機',
    'label-ports':           '連接埠',
    'label-ports-hint':      '（逗號分隔）',

    /* ── Web / DNS fieldset ───────────────────────────────────────────── */
    'legend-web':            'Web / DNS 選項',
    'label-dns-domains':     'DNS 網域',
    'label-dns-domains-hint':'（逗號分隔）',
    'label-record-types':    '記錄類型',
    'label-http-url':        'HTTP URL',
    'label-http-url-hint':   '（選填探測）',

    /* ── SMTP fieldset ────────────────────────────────────────────────── */
    'legend-smtp':           'SMTP 選項',
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
    'legend-ftp':            'FTP / FTPS 選項',
    'label-ftp-user':        '使用者名稱',
    'label-ftp-user-hint':   '（留空 = 匿名）',
    'label-ftp-pass':        '密碼',
    'label-ftp-ssl':         '隱式 FTPS（連接埠 990）',
    'label-ftp-auth-tls':    '顯式 FTPS（AUTH TLS）',
    'label-ftp-list':        '嘗試 PASV + LIST',

    /* ── SFTP fieldset ────────────────────────────────────────────────── */
    'legend-sftp':           'SFTP / SSH 選項',
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
    'btn-run':               '\u25b6 執行診斷',
    'btn-running':           '執行中\u2026',

    /* ── Results & history ────────────────────────────────────────────── */
    'results-title':         '結果',
    'history-title':         '診斷記錄',
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
    'map-target':            '目標',

    /* ── Target select options ────────────────────────────────────────── */
    'opt-web':               'web \u2014 公開 IP 與 DNS',
    'opt-smtp':              'smtp \u2014 郵件伺服器',
    'opt-imap':              'imap \u2014 IMAP 伺服器',
    'opt-pop':               'pop \u2014 POP3 伺服器',
    'opt-ftp':               'ftp \u2014 FTP / FTPS',
    'opt-sftp':              'sftp \u2014 SFTP / SSH',

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
  },
};

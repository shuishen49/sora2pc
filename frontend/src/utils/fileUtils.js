/**
 * File Naming Utilities
 * Ported from legacy static/js/generate.js
 */

export const padNum = (n, width = 2) => {
    const v = Math.max(0, parseInt(String(n ?? '0'), 10) || 0);
    const s = String(v);
    return s.length >= width ? s : `${'0'.repeat(width)}${s}`.slice(-width);
};

export const sanitizeFilename = (name, fallback = 'download') => {
    let s = String(name || '').trim();
    if (!s) return fallback;
    // Remove control characters (eslint-disable-next-line no-control-regex)
    s = s.replace(/[\u0000-\u001f\u007f]/g, '');
    // Replace Windows forbidden chars: \ / : * ? " < > |
    s = s.replace(/[\\/:*?"<>|]/g, '-');
    // Normalize whitespace
    s = s.replace(/\s+/g, ' ').trim();
    // Remove trailing dots/spaces
    s = s.replace(/[. ]+$/g, '');
    if (!s) return fallback;
    // Limit length
    if (s.length > 120) s = s.slice(0, 120).trim();
    return s || fallback;
};

export const mediaExtFromUrl = (url, type = 'video') => {
    const s = String(url || '');
    const m = s.match(/\.([a-zA-Z0-9]{2,6})(?:[?#]|$)/);
    const ext = m ? String(m[1]).toLowerCase() : '';
    const ok = new Set(['mp4', 'mov', 'm4v', 'webm', 'gif', 'png', 'jpg', 'jpeg', 'webp']);
    if (ok.has(ext)) return ext;
    return type === 'image' ? 'png' : 'mp4';
};

export const buildDownloadFilename = (task, url, type = 'video') => {
    // 1. Determine Extension
    const ty = String(type || '').toLowerCase() === 'image' ? 'image' : 'video';
    const ext = mediaExtFromUrl(url, ty);
    const id = task?.id;

    // 2. Storyboard Logic
    if (task && task.storyboard) {
      const sb = task.storyboard || {};
      const idx = parseInt(String(sb.idx || '0'), 10) || 0;
      const total = parseInt(String(sb.total || '0'), 10) || 0;

      const titleRaw = String(sb.title || '分镜').trim();
      const title = sanitizeFilename(titleRaw, '分镜');

      // ex: 分镜01 or 分镜01of10
      const shotPart = idx ? `分镜${padNum(idx, 2)}${total ? `of${padNum(total, 2)}` : ''}` : `分镜`;
      const idPart = id ? `T${id}` : '';

      const parts = [title, shotPart, idPart].filter(Boolean);
      return `${sanitizeFilename(parts.join('_'), '分镜')}.${ext}`;
    }

    // 3. Normal Task Logic
    const prefix = id ? `任务${id}` : `${ty === 'image' ? '图片' : '视频'}`;
    const hintRaw = task && task.prompt ? String(task.prompt).trim() : '';
    // Take first 26 chars of prompt as hint
    const hint = hintRaw ? sanitizeFilename(hintRaw.slice(0, 26), '') : '';

    return `${sanitizeFilename(hint ? `${prefix}_${hint}` : prefix, prefix)}.${ext}`;
};

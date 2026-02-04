/**
 * Enhances upstream API errors with user-friendly messages.
 * Ported from legacy static/js/generate.js
 */
export const humanizeUpstreamError = (raw) => {
    const text = String(raw?.message || raw?.error?.message || raw || '').trim();

    // Try to extract JSON from "API request failed: 400 - {json}"
    let inner = null;
    const jsonStart = text.indexOf('{');
    if (jsonStart >= 0) {
      const maybe = text.slice(jsonStart);
      try {
        inner = JSON.parse(maybe);
      } catch (_) {
        inner = null;
      }
    }

    const err = inner && inner.error ? inner.error : raw && raw.error ? raw.error : null;
    const code = err && err.code ? String(err.code) : '';
    const param = err && err.param ? String(err.param) : '';
    const msg = err && err.message ? String(err.message) : '';
    const merged = (msg || text || '').trim();

    // Region restriction
    const ccFromText = (() => {
      const m = merged.match(/\(([A-Za-z]{2})\)/);
      return m ? m[1] : '';
    })();

    if (
      code === 'unsupported_country_code' ||
      /not available in your country/i.test(merged) ||
      /国家\/地区不可用|地区不可用|Sora.*不可用/i.test(merged)
    ) {
      const cc = param || ccFromText || '未知';
      return {
        type: 'error',
        title: '地区限制',
        message: `Sora 在你当前网络出口地区不可用（${cc}）。\n解决：切换代理/机房到支持地区后再试。`
      };
    }

    // Cloudflare challenge
    if (/Just a moment|Enable JavaScript and cookies to continue|__cf_bm|cloudflare/i.test(text)) {
      return {
        type: 'error',
        title: 'Cloudflare 拦截',
        message: '触发 Cloudflare 风控拦截。\n解决：更换更“干净”的出口 IP/代理，或降低并发与请求频率。'
      };
    }

    // Content Policy
    if (
      /Content Policy Violation/i.test(merged) ||
      /may violate our content policies/i.test(merged) ||
      /content policies?/i.test(merged) && /violate|violation/i.test(merged) ||
      /内容.*(政策|审核|审查)/.test(merged) ||
      /审核未通过|审查未通过|内容不合规|内容违规/.test(merged)
    ) {
        return {
            type: 'warn',
            title: '内容审查拦截',
            message: '生成内容可能违反了相关政策（如NSFW、暴力等）。请修改提示词后重试。'
        }
    }

    // Default fallback
    if (merged) {
      return {
        type: /warn|limit|blocked|guardrail|违规|不支持|限制/i.test(merged) ? 'warn' : 'error',
        title: '生成失败',
        message: merged
      };
    }

    return { type: 'error', title: '生成失败', message: '未知错误（上游未返回可读信息）' };
  };

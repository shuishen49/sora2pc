/**
 * JWT 解析工具（仅 Base64URL 解码，不校验签名，本地展示用）
 * 从 Access Token (JWT) 中提取 email、exp 等
 */

function base64UrlDecode(str) {
  if (!str || typeof str !== 'string') return null
  try {
    let s = str.replace(/-/g, '+').replace(/_/g, '/')
    while (s.length % 4) s += '='
    return decodeURIComponent(
      atob(s)
        .split('')
        .map((c) => '%' + ('00' + c.charCodeAt(0).toString(16)).slice(-2))
        .join('')
    )
  } catch {
    return null
  }
}

/**
 * 解析 JWT，返回 { header, payload } 或 null
 * @param {string} token - JWT 字符串 (header.payload.signature)
 */
export function parseJwt(token) {
  if (!token || typeof token !== 'string') return null
  const parts = token.trim().split('.')
  if (parts.length !== 3) return null
  try {
    const headerJson = base64UrlDecode(parts[0])
    const payloadJson = base64UrlDecode(parts[1])
    if (!headerJson || !payloadJson) return null
    return {
      header: JSON.parse(headerJson),
      payload: JSON.parse(payloadJson),
    }
  } catch {
    return null
  }
}

/**
 * 从 JWT 中提取邮箱（兼容常见 claim 名）
 * @param {string} token - JWT 字符串
 * @returns {string|null} 邮箱或 null
 */
// OpenAI JWT 中邮箱在 payload["https://api.openai.com/profile"].email
const OPENAI_PROFILE_KEY = 'https://api.openai.com/profile'

export function getEmailFromJwt(token) {
  const parsed = parseJwt(token)
  if (!parsed || !parsed.payload) return null
  const p = parsed.payload
  const profile = p[OPENAI_PROFILE_KEY]
  const email =
    p.email ??
    (profile && typeof profile.email === 'string' ? profile.email : null) ??
    p.preferred_username ??
    p.unique_name ??
    p.upn ??
    (typeof p.sub === 'string' && p.sub.includes('@') ? p.sub : null) ??
    null
  return email
}

/**
 * 从 JWT 中提取过期时间戳（秒）
 * @param {string} token - JWT 字符串
 * @returns {number|null} Unix 秒或 null
 */
export function getExpFromJwt(token) {
  const parsed = parseJwt(token)
  if (!parsed || !parsed.payload) return null
  const exp = parsed.payload.exp
  return typeof exp === 'number' ? exp : null
}

/**
 * 检查 JWT 是否已过期
 * @param {string} token - JWT 字符串
 * @returns {boolean}
 */
export function isJwtExpired(token) {
  const exp = getExpFromJwt(token)
  if (exp == null) return false
  return Date.now() / 1000 > exp
}

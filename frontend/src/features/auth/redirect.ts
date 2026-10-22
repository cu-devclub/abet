/** Post-login target: a same-origin relative path, never the login page or a protocol-relative URL. */
export function safeRedirect(path: unknown): string {
  return typeof path === 'string' && path.startsWith('/') && !path.startsWith('//') && !path.startsWith('/login') ? path : '/portfolios'
}

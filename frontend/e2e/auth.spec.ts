import { expect, test } from '@playwright/test'
import { ROLES, statePath } from './env.ts'

test.describe('anonymous', () => {
  test('protected page redirects to /login and offers Google sign-in', async ({ page }) => {
    await page.goto('/portfolios')
    await expect(page).toHaveURL(/\/login/)
    const signIn = page.getByRole('link', { name: 'Sign in with Google' })
    await expect(signIn).toHaveAttribute('href', /\/public\/api\/v1\/auth\/google\/login\?redirect=%2Fportfolios/)
  })

  test('?error=access_denied shows the Access Denied modal', async ({ page }) => {
    await page.goto('/login?error=access_denied')
    await expect(page.getByRole('dialog')).toContainText(/access denied/i)
  })

  test('API without a cookie is 401 and the Google redirect carries PKCE + hd', async ({ request }) => {
    const me = await request.get('/auth/api/v1/me')
    expect(me.status()).toBe(401)
    const login = await request.get('/public/api/v1/auth/google/login?redirect=/courses', { maxRedirects: 0 })
    if (login.status() === 503) test.skip(true, 'GOOGLE_CLIENT_ID not configured')
    expect(login.status()).toBe(302)
    const location = new URL(login.headers()['location'])
    expect(location.hostname).toBe('accounts.google.com')
    expect(location.searchParams.get('code_challenge_method')).toBe('S256')
    expect(location.searchParams.get('hd')).toBeTruthy()
    expect(location.searchParams.get('redirect_uri')).toMatch(/\/public\/api\/v1\/auth\/google\/callback$/)
  })

  test('spoofed X-User-Id header is ignored without a session', async ({ request }) => {
    const res = await request.get('/auth/api/v1/me', { headers: { 'X-User-Id': '00000000-0000-0000-0000-000000000000' } })
    expect(res.status()).toBe(401)
  })
})

test.describe('session', () => {
  test.use({ storageState: statePath('signout') })

  test('signed-in user lands on portfolios and Sign Out returns to /login', async ({ page }) => {
    await page.goto('/')
    await expect(page).toHaveURL(/\/portfolios/)
    // name is empty until the first Google login → the e-mail is shown
    await expect(page.getByText(ROLES.signout, { exact: true }).or(page.getByTitle(ROLES.signout))).toBeVisible()
    await page.getByRole('button', { name: 'Sign Out' }).click()
    await expect(page).toHaveURL(/\/login/)
    const me = await page.request.get('/auth/api/v1/me')
    expect(me.status()).toBe(401) // session revoked server-side, cookie cleared
  })
})

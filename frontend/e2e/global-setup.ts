import { mkdirSync, readFileSync, writeFileSync } from 'node:fs'
import { join } from 'node:path'
import { AUTH_DIR, BASE_URL, REPO_ROOT, ROLES, TMP_DIR, compose, statePath, type RoleKey } from './env.ts'

export default async function globalSetup() {
  mkdirSync(AUTH_DIR, { recursive: true })
  mkdirSync(TMP_DIR, { recursive: true })

  // idempotent dev data (departments, curricula, one user per role, offerings)
  compose(['exec', '-T', 'postgres', 'sh', '-c', 'psql -v ON_ERROR_STOP=1 -q -U "$POSTGRES_USER" -d "$MAIN_DB_NAME"'], readFileSync(join(REPO_ROOT, 'deploy', 'dev', 'seed.sql'), 'utf8'))

  // one session cookie per role (same as `make dev-session email=...`)
  const url = new URL(BASE_URL)
  for (const role of Object.keys(ROLES) as RoleKey[]) {
    const out = compose(['exec', '-T', 'auth-service', 'go', 'run', './cmd/devsession', ROLES[role]]).trim()
    const match = /^abet_session=(.+)$/m.exec(out)
    if (!match) throw new Error(`devsession did not print a cookie for ${role}: ${out}`)
    const state = {
      cookies: [
        {
          name: 'abet_session',
          value: match[1],
          domain: url.hostname,
          path: '/',
          expires: Math.floor(Date.now() / 1000) + 6 * 3600,
          httpOnly: true,
          secure: false,
          sameSite: 'Lax' as const,
        },
      ],
      origins: [],
    }
    writeFileSync(statePath(role), JSON.stringify(state))
  }

}

import { execFileSync } from 'node:child_process'
import { readFileSync } from 'node:fs'
import { dirname, join, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const here = dirname(fileURLToPath(import.meta.url))

export const REPO_ROOT = resolve(here, '..', '..')
export const AUTH_DIR = join(here, '.auth')
export const TMP_DIR = join(here, '.tmp')
export const FIXTURE_DIR = join(REPO_ROOT, 'main-service', 'internal', 'core', 'validation', 'testdata', 'package')
export const BASE_URL = process.env.BASE_URL ?? 'http://localhost:8000'
export const MAIN_API = '/auth/api/v1'

/** Seeded users (deploy/dev/seed.sql). */
export const ROLES = {
  super: 'super1@chula.ac.th',
  admin: 'admin1@chula.ac.th', // scope: department CS
  admin2: 'admin2@chula.ac.th', // scope: curriculum MATH-B only
  instructor: 'instructor1@chula.ac.th',
  instructor2: 'instructor2@chula.ac.th',
  staff: 'staff1@chula.ac.th',
  signout: 'instructor1@chula.ac.th', // its own session: the sign-out test revokes it
} as const

export type RoleKey = keyof typeof ROLES

export function statePath(role: RoleKey): string {
  return join(AUTH_DIR, `${role}.json`)
}

/** Dev compose project (`<COMPOSE_PROJECT_NAME>-dev`, same rule as the Makefile). Only that one key of .env is read. */
export function devProject(): string {
  let name = 'abet'
  try {
    const m = /^COMPOSE_PROJECT_NAME=(.*)$/m.exec(readFileSync(join(REPO_ROOT, '.env'), 'utf8'))
    if (m) name = m[1].replace(/["'\s]/g, '') || name
  } catch {
    // no .env: default
  }
  return `${name}-dev`
}

/** `docker compose` against the dev stack. */
export function compose(args: string[], input?: string | Buffer): string {
  return execFileSync('docker', ['compose', '-p', devProject(), '-f', 'compose.yaml', '-f', 'compose.dev.yaml', ...args], {
    cwd: REPO_ROOT,
    input,
    encoding: 'utf8',
    maxBuffer: 16 * 1024 * 1024,
  })
}

/** Object keys under `prefix` in the MinIO bucket (mc from the minio-init image on the dev network; root creds come from the container env). */
export function mcList(prefix: string): string[] {
  const out = compose(
    ['run', '--rm', '-T', '--no-deps', '--entrypoint', 'sh', 'minio-init', '-c',
      `mc alias set local http://minio:9000 "$MINIO_ROOT_USER" "$MINIO_ROOT_PASSWORD" >/dev/null && mc ls --recursive "local/$MINIO_BUCKET/${prefix}"`],
  )
  // lines look like: [2026-10-10 07:22:40 UTC]  1.2MiB STANDARD uploads/<sid>.zip  (paths are relative to the listed prefix)
  return out.split('\n').map((l) => /\S+\s*$/.exec(l.trim())?.[0] ?? '').filter((k) => k && !k.startsWith('Container') && !k.startsWith('['))
}

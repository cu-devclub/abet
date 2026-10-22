import '@testing-library/jest-dom/vitest'
import { configure } from '@testing-library/react'

// findBy*/waitFor default to 1 s, which flakes on a loaded machine
configure({ asyncUtilTimeout: 10_000 })

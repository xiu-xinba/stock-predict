import { existsSync, readdirSync, readFileSync } from 'node:fs'
import { extname, join, resolve } from 'node:path'
import { describe, expect, it } from 'vitest'

const srcRoot = resolve(process.cwd(), 'src')
const sourceExtensions = new Set(['.ts', '.tsx', '.vue'])

function sourceFiles(directory: string): string[] {
  return readdirSync(directory, { withFileTypes: true }).flatMap((entry) => {
    const path = join(directory, entry.name)
    if (entry.isDirectory()) {
      return entry.name === '__tests__' ? [] : sourceFiles(path)
    }
    return sourceExtensions.has(extname(entry.name)) ? [path] : []
  })
}

describe('frontend architecture', () => {
  it('keeps the legacy source directories removed', () => {
    const legacyDirectories = [
      'api',
      'components',
      'composables',
      'router',
      'stores',
      'types',
      'utils',
      'views',
      '__tests__',
    ]

    for (const directory of legacyDirectories) {
      expect(existsSync(join(srcRoot, directory)), directory).toBe(false)
    }
  })

  it('keeps shared independent from app and business features', () => {
    const violations = sourceFiles(join(srcRoot, 'shared')).filter((file) =>
      /@\/(?:app|features)\//.test(readFileSync(file, 'utf8')),
    )

    expect(violations).toEqual([])
  })

  it('requires cross-feature imports to use the target public entry', () => {
    const featuresRoot = join(srcRoot, 'features')
    const violations: string[] = []

    for (const feature of readdirSync(featuresRoot, { withFileTypes: true })) {
      if (!feature.isDirectory()) continue

      for (const file of sourceFiles(join(featuresRoot, feature.name))) {
        const source = readFileSync(file, 'utf8')
        for (const match of source.matchAll(/['"]@\/features\/([^/'"]+)(\/[^'"]+)?['"]/g)) {
          const targetFeature = match[1]
          const internalPath = match[2]
          if (targetFeature !== feature.name && internalPath) {
            violations.push(`${file}: @/features/${targetFeature}${internalPath}`)
          }
        }
      }
    }

    expect(violations).toEqual([])
  })
})

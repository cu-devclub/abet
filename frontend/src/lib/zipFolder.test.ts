import JSZip from 'jszip'
import { buildZip, normalizePath, shouldSkipPath, toFolderFiles, topLevelFolder, zipFileName, zipFromFileList } from './zipFolder'

const file = (name: string, content = 'x') => new File([content], name.split('/').pop() ?? name)

describe('normalizePath', () => {
  it('strips leading slashes, backslashes and dot segments', () => {
    expect(normalizePath('/Folder/sub\\a.csv')).toBe('Folder/sub/a.csv')
    expect(normalizePath('./Folder//a.csv')).toBe('Folder/a.csv')
  })
})

describe('shouldSkipPath', () => {
  it('skips OS metadata files and __MACOSX directories', () => {
    expect(shouldSkipPath('Folder/.DS_Store')).toBe(true)
    expect(shouldSkipPath('Folder/sub/Thumbs.db')).toBe(true)
    expect(shouldSkipPath('Folder/thumbs.DB')).toBe(true)
    expect(shouldSkipPath('__MACOSX/Folder/._a.csv')).toBe(true)
    expect(shouldSkipPath('Folder/__MACOSX/x')).toBe(true)
  })

  it('keeps normal files', () => {
    expect(shouldSkipPath('Folder/a.csv')).toBe(false)
    expect(shouldSkipPath('Folder/.gitkeep')).toBe(false)
  })
})

describe('topLevelFolder / zipFileName', () => {
  it('uses the first path segment of nested files', () => {
    expect(topLevelFolder(['MyFolder/a.csv', 'MyFolder/b/c.csv'])).toBe('MyFolder')
    expect(zipFileName('MyFolder')).toBe('MyFolder.zip')
  })

  it('falls back when there is no folder', () => {
    expect(topLevelFolder(['a.csv'])).toBeNull()
    expect(zipFileName(null)).toBe('portfolio.zip')
  })
})

describe('toFolderFiles', () => {
  it('drops skipped paths and normalises the rest', () => {
    const result = toFolderFiles([
      { path: '/F/a.csv', file: file('a.csv') },
      { path: '/F/.DS_Store', file: file('.DS_Store') },
      { path: '/__MACOSX/F/._a.csv', file: file('._a.csv') },
    ])
    expect(result.map((r) => r.path)).toEqual(['F/a.csv'])
  })
})

describe('buildZip', () => {
  it('keeps the top-level folder name and names the zip after it', async () => {
    const zipFile = await buildZip([
      { path: 'CourseX/data/a.csv', file: file('a.csv', 'hello') },
      { path: 'CourseX/.DS_Store', file: file('.DS_Store') },
    ])
    expect(zipFile.name).toBe('CourseX.zip')
    const zip = await JSZip.loadAsync(await zipFile.arrayBuffer())
    expect(Object.keys(zip.files).filter((n) => !zip.files[n].dir)).toEqual(['CourseX/data/a.csv'])
  })

  it('rejects an empty folder', async () => {
    await expect(buildZip([{ path: 'F/.DS_Store', file: file('.DS_Store') }])).rejects.toThrow(/no files/)
  })
})

describe('zipFromFileList', () => {
  it('uses webkitRelativePath for entry names', async () => {
    const f = file('a.csv', 'data')
    Object.defineProperty(f, 'webkitRelativePath', { value: 'Picked/sub/a.csv' })
    const zipFile = await zipFromFileList([f])
    expect(zipFile.name).toBe('Picked.zip')
    const zip = await JSZip.loadAsync(await zipFile.arrayBuffer())
    expect(zip.file('Picked/sub/a.csv')).not.toBeNull()
  })
})

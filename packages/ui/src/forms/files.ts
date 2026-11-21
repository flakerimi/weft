/**
 * Files for FileUpload. Nothing here reads a file's contents.
 *
 * - `accepts(file, accept)`: whether a file matches an `accept` list as the
 *   browser reads it: MIME types (`image/png`), wildcards (`image/*`) and
 *   extensions (`.pdf`), comma-separated; an empty list accepts anything.
 *   The file dialog only suggests the list (people can choose any file),
 *   and drops and pastes ignore it, so every file is checked.
 * - `formatSize(bytes, locale)`: a size in the locale (`1.5 kB`, `2,5 MB`),
 *   in powers of 1000 as file managers show them.
 * - `isImage(file)`: an image, whose preview is an object URL.
 */
export function accepts(file: File, accept: string | undefined): boolean {
  const types = (accept ?? '').split(',').map((t) => t.trim().toLowerCase()).filter(Boolean)
  if (!types.length) return true
  const name = file.name.toLowerCase()
  const mime = file.type.toLowerCase()
  return types.some((t) => {
    if (t === '*' || t === '*/*') return true
    if (t.startsWith('.')) return name.endsWith(t)
    if (t.endsWith('/*')) return mime.startsWith(t.slice(0, -1))
    return mime === t
  })
}

const UNITS = ['byte', 'kilobyte', 'megabyte', 'gigabyte', 'terabyte'] as const

export function formatSize(bytes: number, locale: string): string {
  let i = 0
  let n = bytes
  while (n >= 1000 && i < UNITS.length - 1) {
    n /= 1000
    i++
  }
  return new Intl.NumberFormat(locale, { style: 'unit', unit: UNITS[i], unitDisplay: i === 0 ? 'narrow' : 'short', maximumFractionDigits: 1 }).format(n)
}

export const isImage = (file: File) => file.type.startsWith('image/')

/** Sends one file: report progress (0 to 1) through `onProgress`, stop when `signal` aborts, resolve when done, reject on failure. */
export type Upload = (file: File, context: { signal: AbortSignal; onProgress: (fraction: number) => void }) => Promise<unknown>
/** A file refused, and why: its `type` is not accepted, its `size` is over `max-size`, or it is past the `count` allowed. */
export interface FileRejection {
  file: File
  reason: 'type' | 'size' | 'count'
}
/** The words FileUpload says; give some or all to translate them. A file's name follows the action words ("Remove photo.png"). */
export interface FileUploadLabels {
  dropOne: string
  dropMany: string
  browse: string
  chooseOne: string
  chooseMany: string
  files: string
  remove: string
  cancel: string
  retry: string
  uploading: string
  failed: string
  added: string
  removed: string
  uploaded: string
  rejected: string
}
/** A listed file and how its upload stands. */
export interface FileEntry {
  file: File
  status: 'idle' | 'uploading' | 'done' | 'error'
  /** 0 to 1 while uploading, null while unknown. */
  progress: number | null
  error: unknown
}

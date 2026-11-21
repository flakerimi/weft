/**
 * useToast(): the page's queue of toasts, which a `Toaster` shows.
 *
 *     const toast = useToast()
 *     toast.add({ title: 'Saved', tone: 'success' })
 *
 * - `add(message)` puts a toast at the end and returns it, with its `id`
 *   (given, or made); a message with the id of one already queued replaces it.
 * - `update(id, patch)` changes a queued toast; `remove(id)` takes it away
 *   and calls its `onClose`; `clear()` empties the queue.
 * - `toasts` is the queue, reactive, oldest first.
 *
 * A toast leaves on its own after `duration` ms, by default the theme's
 * `--weft-duration-toast`; 0 keeps it until it is closed. Replacing it
 * (add again with its id, or update) starts its time again and has it
 * announced again: `revision` counts those replacements. The queue is one
 * per page, so add toasts from the browser (event handlers): on the server
 * `add` does nothing, as a module shared by every request must not keep one
 * request's toasts for the next.
 */
import { readonly, shallowRef, type Ref } from 'vue'
import type { Tone } from '../types.ts'

export interface ToastAction {
  /** The button's text. */
  label: string
  /** Run on press; the toast closes after it. */
  onClick?: (event: MouseEvent) => void
}

export interface ToastMessage {
  id: string
  /** How many times it has been replaced (added again with its id, or updated); set by the queue. */
  revision: number
  title?: string
  description?: string
  /** The colour and what it says: `success`, `danger`, … */
  tone?: Tone
  /** A Lucide icon before the text. */
  icon?: string
  /** ms before it leaves on its own; default `--weft-duration-toast`; 0 stays until closed. */
  duration?: number
  /** Show the close button (default true). */
  close?: boolean
  /** How it is announced: `foreground` at once (assertive), `background` politely; default foreground for `danger`, background otherwise. */
  type?: 'foreground' | 'background'
  /** Show its remaining time as a bar (default true). */
  progress?: boolean
  /** Buttons after the text. */
  actions?: ToastAction[]
  /** Called when it leaves, however it leaves. */
  onClose?: () => void
}

export type ToastInput = Omit<ToastMessage, 'id' | 'revision'> & { id?: string }

const queue = shallowRef<readonly ToastMessage[]>([])
let next = 0

function add(input: ToastInput): ToastMessage {
  const id = input.id ?? `toast-${++next}`
  const at = queue.value.findIndex((t) => t.id === id)
  const message: ToastMessage = { ...input, id, revision: at < 0 ? 0 : queue.value[at]!.revision + 1 }
  if (typeof document === 'undefined') return message
  queue.value = at < 0 ? [...queue.value, message] : queue.value.map((t, i) => (i === at ? message : t))
  return message
}

function update(id: string, patch: Partial<ToastInput>): void {
  queue.value = queue.value.map((t) => (t.id === id ? { ...t, ...patch, id, revision: t.revision + 1 } : t))
}

function remove(id: string): void {
  const found = queue.value.find((t) => t.id === id)
  if (!found) return
  queue.value = queue.value.filter((t) => t !== found)
  found.onClose?.()
}

function clear(): void {
  const all = queue.value
  queue.value = []
  for (const t of all) t.onClose?.()
}

export interface Toasts {
  toasts: Readonly<Ref<readonly ToastMessage[]>>
  add: (message: ToastInput) => ToastMessage
  update: (id: string, patch: Partial<ToastInput>) => void
  remove: (id: string) => void
  clear: () => void
}

const api: Toasts = { toasts: readonly(queue) as Readonly<Ref<readonly ToastMessage[]>>, add, update, remove, clear }

export function useToast(): Toasts {
  return api
}

/** A CSS time (`4000ms`, `1.5s`) in ms, or undefined when it is not one. */
export function cssTime(value: string): number | undefined {
  const m = /^\s*([\d.]+)(ms|s)\s*$/.exec(value)
  if (!m) return undefined
  return parseFloat(m[1]!) * (m[2] === 's' ? 1000 : 1)
}

<script setup vapor lang="ts">
// Files to send with a form. On pointer it is a drop zone with a Browse
// button: files dropped on the zone, pasted while focus is inside, or chosen
// in the file dialog (the button, or a press on the zone) are added. On
// touch it is one button, which opens the platform's chooser: the camera,
// the photo library or files, as `accept` and `capture` allow. Below, in both
// forms, each file is listed with its name and size in the locale, an image
// with a preview (an object URL, revoked when the file goes: no file is
// ever read), and a remove button. With `upload`, each file added is sent
// through it, with its progress, a cancel (which aborts its signal) and a
// retry after a failure; without it, the files are only collected for
// v-model and for the form, whose submit sends them from the hidden file
// input. Files of another type, too large or past `max-files` are refused
// and reported (`reject`). What happens is said in a polite live region.
import { computed, markRaw, nextTick, onScopeDispose, reactive, ref, toRaw, useTemplateRef, watch } from 'vue'
import Button from '../basics/Button.vue'
import Progress from '../basics/Progress.vue'
import CloseButton from '../overlays/CloseButton.vue'
import { useWeft } from '../adaptive/context.ts'
import { focus } from '../headless/dom.ts'
import { useControllable } from '../headless/useControllable.ts'
import { useId } from '../headless/useId.ts'
import { useLocale } from '../adaptive/locale.ts'
import { useControlAttrs, useField } from './field.ts'
import { joinIds } from './dom.ts'
import { accepts, formatSize, isImage, type FileEntry, type FileRejection, type FileUploadLabels, type Upload } from './files.ts'
import type { Size } from '../types.ts'
import '../styles/base.css'
import '../styles/file-upload.css'

defineOptions({ inheritAttrs: false })


const props = withDefaults(
  defineProps<{
    /** The files, bound with `v-model`: an array with `multiple`, else one File or null. */
    modelValue?: File[] | File | null
    /** Several files; without it a new file replaces the one there. */
    multiple?: boolean
    /** The types allowed, as the native attribute: MIME types, `image/*`, extensions (`.pdf`), comma-separated. */
    accept?: string
    /** On a phone, go straight to a camera: `environment` the back one, `user` the front one. */
    capture?: 'user' | 'environment'
    /** The largest file allowed, in bytes. */
    maxSize?: number
    /** The most files allowed, with `multiple`. */
    maxFiles?: number
    /** Send each file added: `(file, { signal, onProgress }) => Promise`. Without it the files are only collected. */
    upload?: Upload
    /** The zone's text, in place of "Drop files here". */
    label?: string
    /** A hint under the zone's text (the types and sizes allowed); it describes the button. */
    description?: string
    /** The button's text, in place of "Browse" (on touch "Choose files"). */
    buttonLabel?: string
    /** List the files below. */
    preview?: boolean
    /** A drop zone on pointer; false leaves only the button. */
    dropzone?: boolean
    /** A press anywhere on the zone opens the file dialog. */
    interactive?: boolean
    /** Unavailable: nothing can be added or removed. */
    disabled?: boolean
    /** A file is required (the native input's `required`). */
    required?: boolean
    /** The form field name the files submit under; a FormField's name applies when not set. */
    name?: string
    /** The button's id; inside a FormField the field's id is used, so its label points at the button. */
    id?: string
    /** `sm`, `md` or `lg`. On touch the button is at least 44 px tall. */
    size?: Size
    /** The BCP 47 locale of the sizes; a WeftProvider's applies when not set, else en-US. */
    locale?: string
    /** The words it says, to translate them. */
    labels?: Partial<FileUploadLabels>
  }>(),
  { modelValue: undefined, preview: true, dropzone: true, interactive: true },
)

const emit = defineEmits<{
  /** The files changed. */
  'update:modelValue': [value: File[] | File | null]
  /** Files were added or removed: the files now. */
  change: [files: File[]]
  /** Files were refused, and why. */
  reject: [rejections: FileRejection[]]
  /** A file was removed. */
  remove: [file: File]
  /** A file was uploaded: what `upload` resolved with. */
  uploaded: [file: File, result: unknown]
  /** A file failed to upload: what `upload` rejected with. */
  error: [file: File, error: unknown]
}>()


defineSlots<{
  /** The zone's icon. */
  leading?: () => unknown
  /** The zone's text. */
  label?: () => unknown
  /** The hint under it. */
  description?: () => unknown
  /** A listed file's whole row. */
  file?: (props: FileEntry & { remove: () => void; retry: () => void }) => unknown
}>()

const LABELS: FileUploadLabels = {
  dropOne: 'Drop a file here',
  dropMany: 'Drop files here',
  browse: 'Browse',
  chooseOne: 'Choose a file',
  chooseMany: 'Choose files',
  files: 'Files',
  remove: 'Remove',
  cancel: 'Cancel upload of',
  retry: 'Retry',
  uploading: 'Uploading',
  failed: 'Upload failed',
  added: 'added',
  removed: 'removed',
  uploaded: 'uploaded',
  rejected: 'not added',
}
const t = computed(() => ({ ...LABELS, ...props.labels }))
const f = useField(props)
const { own, box, describedBy } = useControlAttrs(f)
const model = useControllable(props, 'modelValue', emit, () => (props.multiple ? [] : null))
const weft = useWeft()
const touch = computed(() => weft.input.value === 'touch')
const zone = computed(() => !touch.value && props.dropzone)
const locale = useLocale(() => props.locale)
const uid = useId('file-upload')
const browseId = computed(() => f.id.value ?? `${uid}-browse`)
const hintId = `${uid}-hint`
const input = useTemplateRef<HTMLInputElement>('input')
// Files as given, never through a reactive proxy, so they compare by identity.
const files = computed<File[]>(() => (Array.isArray(model.value) ? model.value : model.value ? [model.value] : []).map((file) => toRaw(file)))

// The listed files, one entry per file in the value, each with its preview and upload.
interface Entry extends FileEntry {
  id: number
  url: string | null
  controller: AbortController | null
}
const entries = reactive<Entry[]>([])
/** An entry's file as given (a test DOM may hand files to reactivity, which would wrap them). */
const fileOf = (e: Entry) => toRaw(e.file)
let ids = 0
const canPreview = () => typeof URL !== 'undefined' && typeof URL.createObjectURL === 'function' && typeof document !== 'undefined'
function drop(e: Entry) {
  e.controller?.abort()
  e.controller = null
  if (e.url) URL.revokeObjectURL(e.url)
  e.url = null
}
/** Make the entries the files of the value, in its order: new files get an entry, gone ones lose theirs. */
function reconcile(list: File[]) {
  for (const e of entries) if (!list.includes(fileOf(e))) drop(e)
  const next = list.map(
    (file) =>
      entries.find((e) => fileOf(e) === file) ?? {
        id: ++ids,
        file,
        url: props.preview && isImage(file) && canPreview() ? URL.createObjectURL(file) : null,
        status: 'idle' as const,
        progress: null,
        error: null,
        controller: null,
      },
  )
  entries.splice(0, entries.length, ...next)
}
watch(files, reconcile, { immediate: true })
onScopeDispose(() => entries.forEach(drop))

// The hidden input holds the files too, so a native submit sends them.
watch(
  [files, input],
  ([list, el]) => {
    if (!el) return
    try {
      const dt = new DataTransfer()
      for (const file of list) dt.items.add(file)
      el.files = dt.files
    } catch {
      // A browser without a DataTransfer constructor keeps what its dialog chose.
    }
  },
  { flush: 'post' },
)

// What happens, said politely; the same words again are said again.
const said = ref('')
function say(words: string) {
  said.value = said.value === words ? `${words} ` : words
}
const listOf = (list: File[]) => list.map((file) => file.name).join(', ')

function write(list: File[]) {
  model.value = props.multiple ? list : (list[0] ?? null)
  reconcile(list)
  emit('change', list)
  f.input()
}

function add(incoming: File[]) {
  if (f.disabled.value || !incoming.length) return
  const accepted: File[] = []
  const rejected: FileRejection[] = []
  const room = props.multiple ? (props.maxFiles ?? Infinity) - files.value.length : 1
  for (const file of incoming) {
    if (!accepts(file, props.accept)) rejected.push({ file, reason: 'type' })
    else if (props.maxSize !== undefined && file.size > props.maxSize) rejected.push({ file, reason: 'size' })
    else if (accepted.length >= room) rejected.push({ file, reason: 'count' })
    else accepted.push(file)
  }
  const words: string[] = []
  if (accepted.length) {
    write(props.multiple ? [...files.value, ...accepted] : accepted)
    words.push(`${listOf(accepted)} ${t.value.added}`)
    for (const file of accepted) {
      const e = entries.find((x) => fileOf(x) === file)
      if (e) void start(e)
    }
  }
  if (rejected.length) {
    emit('reject', rejected)
    words.push(`${listOf(rejected.map((r) => r.file))} ${t.value.rejected}`)
  }
  say(words.join('. '))
}

async function start(e: Entry) {
  const upload = props.upload
  if (!upload) return
  e.controller?.abort()
  const controller = markRaw(new AbortController())
  e.controller = controller
  e.status = 'uploading'
  e.progress = null
  e.error = null
  const live = () => !controller.signal.aborted && e.controller === controller
  try {
    const result = await upload(fileOf(e), {
      signal: controller.signal,
      onProgress: (fraction) => {
        if (live()) e.progress = Math.min(1, Math.max(0, fraction))
      },
    })
    if (!live()) return
    e.status = 'done'
    e.progress = 1
    e.controller = null
    emit('uploaded', fileOf(e), result)
    say(`${e.file.name} ${t.value.uploaded}`)
  } catch (error) {
    if (!live()) return
    e.status = 'error'
    e.error = error
    e.controller = null
    emit('error', fileOf(e), error)
    say(`${e.file.name}: ${t.value.failed}`)
  }
}

const root = useTemplateRef<HTMLElement>('root')
function remove(e: Entry) {
  if (f.disabled.value) return
  const at = entries.indexOf(e)
  const hadFocus = !!root.value?.contains(document.activeElement)
  write(files.value.filter((file) => file !== fileOf(e)))
  emit('remove', fileOf(e))
  say(`${e.file.name} ${t.value.removed}`)
  // Focus goes to the next file's remove button, else the last one's, else the button.
  if (hadFocus) {
    void nextTick(() => {
      const buttons = [...(root.value?.querySelectorAll<HTMLElement>('.weft-file-upload__remove') ?? [])]
      focus(buttons[Math.min(at, buttons.length - 1)] ?? document.getElementById(browseId.value))
    })
  }
}

/** Open the file dialog. */
function open() {
  if (f.disabled.value || !input.value) return
  input.value.click()
}
function onPicked(event: Event) {
  const el = event.target as HTMLInputElement
  add([...(el.files ?? [])])
}
function onZoneClick(event: MouseEvent) {
  if (!props.interactive || (event.target as Element).closest('button')) return
  open()
}

// Dragging over the zone: entering and leaving its children counts in and out.
let depth = 0
const dragging = ref(false)
const hasFiles = (event: DragEvent) => {
  const dt = event.dataTransfer
  return !!dt && ([...(dt.types ?? [])].includes('Files') || [...(dt.items ?? [])].some((item) => item.kind === 'file'))
}
function onDragenter(event: DragEvent) {
  if (f.disabled.value || !hasFiles(event)) return
  event.preventDefault()
  depth++
  dragging.value = true
}
function onDragover(event: DragEvent) {
  if (f.disabled.value || !hasFiles(event)) return
  event.preventDefault()
  if (event.dataTransfer) event.dataTransfer.dropEffect = 'copy'
}
function onDragleave() {
  depth = Math.max(0, depth - 1)
  if (!depth) dragging.value = false
}
function onDrop(event: DragEvent) {
  depth = 0
  dragging.value = false
  if (f.disabled.value || !event.dataTransfer?.files.length) return
  event.preventDefault()
  add([...event.dataTransfer.files])
}
function onPaste(event: ClipboardEvent) {
  const list = [...(event.clipboardData?.files ?? [])]
  if (f.disabled.value || !list.length) return
  event.preventDefault()
  add(list)
}

const buttonText = computed(() => props.buttonLabel ?? (touch.value ? (props.multiple ? t.value.chooseMany : t.value.chooseOne) : t.value.browse))
const zoneText = computed(() => props.label ?? (props.multiple ? t.value.dropMany : t.value.dropOne))
const buttonDescribedBy = computed(() => describedBy(zone.value && props.description ? hintId : undefined))
const percent = (e: Entry) => (e.progress === null ? null : Math.round(e.progress * 100))

defineExpose({
  /** Open the file dialog. */
  open,
  /** The hidden file input. */
  input,
})
</script>

<template>
  <div ref="root" class="weft-file-upload" v-bind="{ ...box, ...own }" :data-form="zone ? 'zone' : 'button'" :data-size="f.size.value" :data-disabled="f.disabled.value ? 'true' : undefined" @paste="onPaste">
    <div
      v-if="zone"
      class="weft-file-upload__zone"
      :data-dragging="dragging ? 'true' : undefined"
      :data-invalid="f.invalid.value ? 'true' : undefined"
      :data-interactive="interactive && !f.disabled.value ? 'true' : undefined"
      @click="onZoneClick"
      @dragenter="onDragenter"
      @dragover="onDragover"
      @dragleave="onDragleave"
      @drop="onDrop"
    >
      <span class="weft-file-upload__icon" aria-hidden="true">
        <slot name="leading"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M12 3v12" /><path d="m17 8-5-5-5 5" /><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /></svg></slot>
      </span>
      <p class="weft-file-upload__label"><slot name="label">{{ zoneText }}</slot></p>
      <p v-if="description || $slots.description" :id="hintId" class="weft-file-upload__description"><slot name="description">{{ description }}</slot></p>
      <Button
        class="weft-file-upload__browse"
        variant="outline"
        tone="neutral"
        :size="f.size.value"
        :label="buttonText"
        :id="browseId"
        :aria-labelledby="f.field ? joinIds(f.field.labelId, browseId) : undefined"
        :aria-describedby="buttonDescribedBy"
        :disabled="f.disabled.value"
        @click="open"
      />
    </div>
    <Button
      v-else
      class="weft-file-upload__browse"
      variant="outline"
      tone="neutral"
      :size="f.size.value"
      :label="buttonText"
      :id="browseId"
      :aria-labelledby="f.field ? joinIds(f.field.labelId, browseId) : undefined"
      :aria-describedby="buttonDescribedBy"
      :disabled="f.disabled.value"
      @click="open"
    >
      <template #leading><svg class="weft-file-upload__glyph" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" aria-hidden="true"><path d="M12 3v12" /><path d="m17 8-5-5-5 5" /><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /></svg></template>
    </Button>
    <input
      ref="input"
      class="weft-visually-hidden"
      type="file"
      tabindex="-1"
      aria-hidden="true"
      :name="f.name.value"
      :accept="accept"
      :multiple="multiple || undefined"
      :capture="capture"
      :required="required || undefined"
      :disabled="f.disabled.value || undefined"
      @change="onPicked"
    />
    <ul v-if="preview && entries.length" class="weft-file-upload__files" :aria-label="t.files">
      <li v-for="e in entries" :key="e.id" class="weft-file-upload__file" :data-status="e.status">
        <slot name="file" :file="fileOf(e)" :status="e.status" :progress="e.progress" :error="e.error" :remove="() => remove(e)" :retry="() => start(e)">
          <img v-if="e.url" class="weft-file-upload__thumb" :src="e.url" alt="" />
          <span v-else class="weft-file-upload__thumb" aria-hidden="true"><svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M15 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V7Z" /><path d="M14 2v4a2 2 0 0 0 2 2h4" /></svg></span>
          <span class="weft-file-upload__meta">
            <span class="weft-file-upload__name">{{ e.file.name }}</span>
            <span class="weft-file-upload__size">{{ formatSize(e.file.size, locale) }}</span>
            <Progress v-if="e.status === 'uploading'" class="weft-file-upload__progress" size="sm" :model-value="percent(e)" :label="`${t.uploading} ${e.file.name}`" />
            <span v-if="e.status === 'error'" class="weft-file-upload__error">{{ t.failed }}</span>
          </span>
          <Button
            v-if="e.status === 'error'"
            class="weft-file-upload__retry"
            variant="ghost"
            tone="neutral"
            size="sm"
            :label="t.retry"
            :aria-label="`${t.retry} ${e.file.name}`"
            :disabled="f.disabled.value"
            @click="start(e)"
          />
          <CloseButton
            class="weft-file-upload__remove"
            :label="e.status === 'uploading' ? `${t.cancel} ${e.file.name}` : `${t.remove} ${e.file.name}`"
            :disabled="f.disabled.value || undefined"
            @click="remove(e)"
          />
        </slot>
      </li>
    </ul>
    <span class="weft-visually-hidden" aria-live="polite">{{ said }}</span>
  </div>
</template>

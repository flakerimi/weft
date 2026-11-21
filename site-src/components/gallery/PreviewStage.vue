<script setup vapor lang="ts">
// The preview stage of a component page: its first example (the slot) inside
// a WeftProvider, under a toolbar that sets the input, surface, theme, scheme
// and width the example is shown with. It is an island (the gallery plugin's
// Stage_<Name>.vue wraps it round the example), and keeps its view in the
// URL query (stage.ts), so a link reproduces it.
//
// The server draws the default view: the example as an app shows it, filling
// the column, in the page's scheme. Without JavaScript that stays, and the
// toolbar is inert. Once hydrated, the stage reads the query and the device
// (its input, its scheme) and the toolbar comes alive. Widths resize a
// container, not an iframe, so the example's useWidth sees them.
import { computed, onMounted, ref } from 'vue'
import { ToggleGroup, WeftProvider, detectInput, type ToggleGroupItem } from '@weft/ui'
import { formatView, parseView, DEFAULT_VIEW, type StageView } from './stage.ts'
import '/.gallery/stage-themes.css'
import '../../styles/stage.css'

defineProps<{
  /** The component shown, for the stage's name. */
  component: string
}>()

const view = ref<StageView>({ ...DEFAULT_VIEW })
const mounted = ref(false)
// What the device does, once known: the toolbar shows it while the view does not force one.
const deviceInput = ref<'pointer' | 'touch' | null>(null)
const pageScheme = ref<'light' | 'dark' | null>(null)

onMounted(() => {
  view.value = parseView(location.search)
  deviceInput.value = detectInput().value
  const dark = matchMedia('(prefers-color-scheme: dark)')
  pageScheme.value = dark.matches ? 'dark' : 'light'
  dark.addEventListener('change', (e) => (pageScheme.value = e.matches ? 'dark' : 'light'))
  mounted.value = true
})

/** Set one axis and write the view into the URL; a press that would leave a group with none pressed is ignored. */
function set<K extends keyof StageView>(key: K, value: unknown) {
  if (value == null) return
  const next = { ...view.value, [key]: value === 'fit' ? undefined : value }
  view.value = next
  history.replaceState(history.state, '', location.pathname + formatView(next) + location.hash)
}

const inputs: ToggleGroupItem[] = [{ label: 'Pointer', value: 'pointer' }, { label: 'Touch', value: 'touch' }]
const surfaces: ToggleGroupItem[] = [{ label: 'App', value: 'app' }, { label: 'Landing', value: 'landing' }]
const themes: ToggleGroupItem[] = [{ label: 'Neutral', value: 'neutral' }, { label: 'Lisa', value: 'lisa' }]
const schemes: ToggleGroupItem[] = [{ label: 'Light', value: 'light' }, { label: 'Dark', value: 'dark' }]
const widths: ToggleGroupItem[] = [
  { label: 'Fit', value: 'fit' },
  { label: 'Phone', value: 390 },
  { label: 'Tablet', value: 820 },
  { label: 'Desktop', value: 1280 },
]

const input = computed(() => view.value.input ?? deviceInput.value)
const scheme = computed(() => view.value.scheme ?? pageScheme.value)
const width = computed(() => view.value.width ?? 'fit')
const canvasStyle = computed(() => (view.value.width ? { inlineSize: `${view.value.width}px` } : undefined))
</script>

<template>
  <section class="gallery-stage" :aria-label="`${component} preview`">
    <div class="gallery-toolbar" :inert="!mounted" :data-ready="mounted ? 'true' : undefined">
      <div class="gallery-toolbar__axis">
        <span class="gallery-toolbar__name" aria-hidden="true">Input</span>
        <ToggleGroup :items="inputs" :model-value="input" size="sm" attached aria-label="Input" @update:model-value="set('input', $event)" />
      </div>
      <div class="gallery-toolbar__axis">
        <span class="gallery-toolbar__name" aria-hidden="true">Surface</span>
        <ToggleGroup :items="surfaces" :model-value="view.surface" size="sm" attached aria-label="Surface" @update:model-value="set('surface', $event)" />
      </div>
      <div class="gallery-toolbar__axis">
        <span class="gallery-toolbar__name" aria-hidden="true">Theme</span>
        <ToggleGroup :items="themes" :model-value="view.theme" size="sm" attached aria-label="Theme" @update:model-value="set('theme', $event)" />
      </div>
      <div class="gallery-toolbar__axis">
        <span class="gallery-toolbar__name" aria-hidden="true">Scheme</span>
        <ToggleGroup :items="schemes" :model-value="scheme" size="sm" attached aria-label="Scheme" @update:model-value="set('scheme', $event)" />
      </div>
      <div class="gallery-toolbar__axis">
        <span class="gallery-toolbar__name" aria-hidden="true">Width</span>
        <ToggleGroup :items="widths" :model-value="width" size="sm" attached aria-label="Width" @update:model-value="set('width', $event)" />
      </div>
    </div>
    <div class="gallery-stage__viewport">
      <div
        class="gallery-canvas"
        :data-theme="view.theme"
        :data-weft-scheme="view.scheme"
        :data-width="view.width"
        :style="canvasStyle"
      >
        <WeftProvider :input="view.input" :surface="view.surface">
          <div class="gallery-example"><slot /></div>
        </WeftProvider>
      </div>
    </div>
  </section>
</template>

<script setup vapor lang="ts">
// The preview stage of a component page: its first example (the slot) inside
// a WeftProvider, under a toolbar that sets the input, surface, theme, scheme
// and width the example is shown with. It is an island (the gallery plugin's
// Stage_<Name>.vue wraps it round the example), and keeps its view in the
// URL query (stage.ts), so a link reproduces it. The toolbar is one row:
// input, surface and scheme, then a button whose Popover holds the theme
// and the width, its label saying which are set. The theme is the site's
// (the header's Theme menu) unless neutral or lisa is forced on the canvas;
// a forced theme's stylesheet is the site's own for it, which the stage
// links when it is picked (the menu's module, /theme-menu.js).
//
// The server draws the default view: the example as an app shows it, filling
// the column, in the page's scheme. Without JavaScript that stays, and the
// toolbar is inert. Once hydrated, the stage reads the query and the device
// (its input, its scheme) and the toolbar comes alive. Widths resize a
// container, not an iframe, so the example's useWidth sees them.
import { computed, onMounted, ref } from 'vue'
import { Button, Popover, ToggleGroup, WeftProvider, detectInput, type ToggleGroupItem } from '@weft/ui'
import { THEME_MODULE } from '../../themes/menu.ts'
import { formatView, parseView, DEFAULT_VIEW, type StageView } from './stage.ts'
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

/** Link a forced theme's stylesheet, once: the site's, through the Theme menu's module. */
function linkTheme(theme: StageView['theme']) {
  if (theme === 'site') return
  void import(/* @vite-ignore */ THEME_MODULE).then((m: { sheet: (t: string) => void }) => m.sheet(theme))
}

onMounted(() => {
  view.value = parseView(location.search)
  linkTheme(view.value.theme)
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
  if (key === 'theme') linkTheme(next.theme)
  view.value = next
  history.replaceState(history.state, '', location.pathname + formatView(next) + location.hash)
}

const inputs: ToggleGroupItem[] = [{ label: 'Pointer', value: 'pointer' }, { label: 'Touch', value: 'touch' }]
const surfaces: ToggleGroupItem[] = [{ label: 'App', value: 'app' }, { label: 'Landing', value: 'landing' }]
const themes: ToggleGroupItem[] = [{ label: 'Site', value: 'site' }, { label: 'Neutral', value: 'neutral' }, { label: 'Lisa', value: 'lisa' }]
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
/** The More button's label: the theme and width when either is set, else what it holds. */
const more = computed(() => {
  const set = [view.value.theme === 'site' ? null : themes.find((t) => t.value === view.value.theme)?.label, view.value.width ? `${view.value.width} px` : null].filter(Boolean)
  return set.length ? set.join(' · ') : 'Theme and width'
})
</script>

<template>
  <section class="gallery-stage" :aria-label="`${component} preview`">
    <div class="gallery-toolbar" role="group" aria-label="Preview settings" :inert="!mounted" :data-ready="mounted ? 'true' : undefined">
      <div class="gallery-toolbar__axis">
        <span class="gallery-toolbar__name" aria-hidden="true">Input</span>
        <ToggleGroup :items="inputs" :model-value="input" size="sm" attached aria-label="Input" @update:model-value="set('input', $event)" />
      </div>
      <div class="gallery-toolbar__axis">
        <span class="gallery-toolbar__name" aria-hidden="true">Surface</span>
        <ToggleGroup :items="surfaces" :model-value="view.surface" size="sm" attached aria-label="Surface" @update:model-value="set('surface', $event)" />
      </div>
      <div class="gallery-toolbar__axis">
        <span class="gallery-toolbar__name" aria-hidden="true">Scheme</span>
        <ToggleGroup :items="schemes" :model-value="scheme" size="sm" attached aria-label="Scheme" @update:model-value="set('scheme', $event)" />
      </div>
      <Popover title="Theme and width" placement="bottom-end">
        <Button class="gallery-toolbar__more" size="sm" variant="outline" tone="neutral" :label="more" trailing-icon="chevron-down" />
        <template #content>
          <div class="gallery-toolbar__more-axes">
            <div class="gallery-toolbar__axis">
              <span class="gallery-toolbar__name" aria-hidden="true">Theme</span>
              <ToggleGroup :items="themes" :model-value="view.theme" size="sm" attached aria-label="Theme" @update:model-value="set('theme', $event)" />
            </div>
            <div class="gallery-toolbar__axis">
              <span class="gallery-toolbar__name" aria-hidden="true">Width</span>
              <ToggleGroup :items="widths" :model-value="width" size="sm" attached aria-label="Width" @update:model-value="set('width', $event)" />
            </div>
          </div>
        </template>
      </Popover>
    </div>
    <div class="gallery-stage__viewport">
      <!-- The example's links name pages of an app that is not here: not prefetched. -->
      <div
        class="gallery-canvas"
        data-weft-no-prefetch
        data-site-picture
        :data-theme="view.theme === 'site' ? undefined : view.theme"
        :data-weft-scheme="view.scheme"
        :data-width="view.width"
        :style="canvasStyle"
      >
        <!-- inset: a Page or an AppShell in the example is a preview, not this page's main or h1. -->
        <WeftProvider :input="view.input" :surface="view.surface" inset>
          <div class="gallery-example"><slot /></div>
        </WeftProvider>
      </div>
    </div>
  </section>
</template>

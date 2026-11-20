import vue from '@vitejs/plugin-vue'
import { loadConfigFromFile, mergeConfig, type ConfigEnv, type InlineConfig, type Plugin, type PluginOption } from 'vite'

async function flatten(option: PluginOption, out: Plugin[] = []): Promise<Plugin[]> {
  const value = await option
  if (Array.isArray(value)) for (const o of value) await flatten(o, out)
  else if (value && 'name' in value) out.push(value as Plugin)
  return out
}

/**
 * The Vite config for work on the site at `root`: the site's own `vite.config`
 * if it has one, with `ours` merged over it. `vue()` is added unless the site's
 * config already has it, since two would compile every SFC twice.
 */
export async function siteConfig(root: string, env: ConfigEnv, ours: InlineConfig): Promise<InlineConfig> {
  const loaded = await loadConfigFromFile(env, undefined, root, 'error')
  const site = loaded?.config ?? {}
  const hasVue = (await flatten(site.plugins)).some((p) => p.name === 'vite:vue')
  const merged = mergeConfig(site, { ...ours, plugins: [...(hasVue ? [] : [vue()]), ...(ours.plugins ?? [])] })
  return { ...merged, root, configFile: false, mode: env.mode }
}

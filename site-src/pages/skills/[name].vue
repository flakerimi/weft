<script lang="ts">
import { contentPaths } from 'virtual:weft/content'

export const getPaths = async () => contentPaths('skills', 'name')
</script>

<script setup vapor lang="ts">
// A skill: skills/<name>/SKILL.md, as the docs are drawn (the sidebar and
// the header say where it is, so it has no breadcrumb), with its own
// heading as the page's title, its name and when to use it under it, and
// how to install it. The page also writes the file as it is, byte for byte,
// at /skills/<name>.md, for an agent to install. It ships no script: the
// install command is text here, with copy buttons on /skills, which keeps
// the page within the docs' 10 KB.
import { Link, Page, useHead } from '@weft/ui'
import { useContent } from 'virtual:weft/content'
import { useFile } from 'virtual:weft/files'
import '@weft/vite/code.css'
import '../../styles/fonts.css'
import '../../styles/docs.css'
import { installCommand } from '../../agents/llms.ts'
import SiteFooter from '../../components/SiteFooter.vue'
import SiteHeader from '../../components/SiteHeader.vue'
import SkillsNav from '../../components/SkillsNav.vue'

const props = defineProps<{ name: string }>()
const skill = useContent('skills', props.name)
const raw = `${skill.path}.md`
useFile(raw, skill.source)

// The skill's own h1 is the page's title; the rest of its body is the text.
const first = /^<h1 id="([^"]*)">(.*?)<\/h1>\n?/.exec(skill.html)
if (!first) throw new Error(`skills: ${skill.file} does not start with a # heading; give the skill one, as its page's title.`)
const heading = first[2]!
// A table scrolls inside a container of its own on a narrow screen, as in the docs.
const body = skill.html.slice(first[0].length).replaceAll('<table>', '<div class="docs-table"><table>').replaceAll('</table>', '</table></div>')
const text = heading.replace(/<[^>]+>/g, '')
useHead({ title: `${text} — Weft skills`, description: skill.frontmatter.description })
const toc = skill.toc
</script>

<template>
  <div class="site">
    <SiteHeader current="agents" :path="skill.path" menu="skills-nav" />
    <div class="docs">
      <SkillsNav :current="skill.path" />
      <Page id="main" class="docs-page">
        <template #header>
          <h1 class="docs-title" v-html="heading" />
          <p class="docs-lead">
            <code>{{ skill.frontmatter.title }}</code> is a Claude Code skill. Its description, which tells an agent when to use it:
            {{ skill.frontmatter.description }}
          </p>
          <p>
            Install it for every project with <code>{{ installCommand([skill.frontmatter.title], '~/.claude/skills') }}</code>,
            read <Link :to="raw">the raw file</Link>, or see <Link to="/skills">every way to install the skills</Link>.
          </p>
        </template>
        <div class="docs-body">
          <nav v-if="toc.length" class="docs-toc" aria-labelledby="docs-toc-title">
            <p id="docs-toc-title">On this page</p>
            <ul>
              <li v-for="entry in toc" :key="entry.id" :data-depth="entry.depth">
                <Link :to="`#${entry.id}`">{{ entry.text }}</Link>
              </li>
            </ul>
          </nav>
          <article class="docs-prose" v-html="body" />
          <nav v-if="skill.prev || skill.next" class="docs-pager" aria-label="Previous and next">
            <Link v-if="skill.prev" :to="skill.prev.path" rel="prev"><span>Previous</span>{{ skill.prev.title }}</Link>
            <Link v-if="skill.next" :to="skill.next.path" rel="next"><span>Next</span>{{ skill.next.title }}</Link>
          </nav>
        </div>
      </Page>
    </div>
    <SiteFooter />
  </div>
</template>

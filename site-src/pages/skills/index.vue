<script setup vapor lang="ts">
// /skills, For agents: Weft's Claude Code skills, each with when to use it
// and its raw file, and how to install them. The page also writes the
// site's files for agents, made from the docs, the components and the
// skills: /llms.txt and /llms-full.txt (agents/llms.ts), and /index.md, the
// page an agent starts from (agents/index-md.ts). It ships no script
// but the install commands' copy buttons.
import { Card, Grid, Island, Link, Page, Section, Stack, useHead } from '@weft/ui'
import { useCollection, useContent } from 'virtual:weft/content'
import { useFile } from 'virtual:weft/files'
import catalogText from '../../../catalog/components.json?raw'
import appAgents from '../../../templates/app/AGENTS.md.tmpl?raw'
import '../../styles/fonts.css'
import '../../styles/docs.css'
import { installCommand, llmsFull, llmsTxt } from '../../agents/llms.ts'
import { indexMd, siteVersion } from '../../agents/index-md.ts'
import type { Catalog } from '../../gallery/model.ts'
import CopyCommand from '../../components/CopyCommand.vue'
import SiteFooter from '../../components/SiteFooter.vue'
import SiteHeader from '../../components/SiteHeader.vue'
import SkillsNav from '../../components/SkillsNav.vue'

const description = 'Claude Code skills for building with Weft, and the files an agent reads.'
useHead({ title: 'For agents — Weft', description })

const skills = useCollection('skills').map((s) => useContent('skills', s.slug))
const docs = useCollection('docs').map((d) => useContent('docs', d.slug))
const input = { docs, skills, components: (JSON.parse(catalogText) as Catalog).components }
useFile('/llms.txt', llmsTxt(input))
useFile('/llms-full.txt', llmsFull(input))
// WEFT_SITE_VERSION is set by scripts/publish-site.sh to the release it publishes; without it the page names no version.
useFile('/index.md', indexMd({ ...input, version: siteVersion(process.env.WEFT_SITE_VERSION), appAgents }))

const names = skills.map((s) => s.frontmatter.title)
const commands = [
  { title: 'For every project', text: 'Into ~/.claude/skills, for every project on this machine.', command: installCommand(names, '~/.claude/skills') },
  { title: 'For one project', text: 'Into the project’s .claude/skills, to commit with it.', command: installCommand(names, '.claude/skills') },
  { title: 'From a clone of Weft', text: 'The same files, from the repository’s skills folder.', command: 'mkdir -p .claude/skills && cp -R skills/weft-* .claude/skills/' },
]
</script>

<template>
  <div class="site">
    <SiteHeader current="agents" path="/skills" menu="skills-nav" :themes="false" />
    <div class="docs">
      <SkillsNav current="/skills" />
      <Page id="main" class="docs-page">
        <template #header>
          <h1 class="docs-title">For agents</h1>
          <p class="docs-lead">{{ description }}</p>
        </template>
        <div class="docs-prose">
          <Section title="Skills">
            <Grid :cols="{ narrow: 1, regular: 2, wide: 2 }">
              <Card v-for="s in skills" :key="s.slug">
                <template #header><Link :to="s.path">{{ s.frontmatter.title }}</Link></template>
                <Stack :gap="2">
                  <p>{{ s.frontmatter.description }}</p>
                  <p><Link :to="`${s.path}.md`">Raw SKILL.md</Link></p>
                </Stack>
              </Card>
            </Grid>
          </Section>
          <Section title="Install them in Claude Code">
            <p>A skill is a folder with a SKILL.md. Claude Code picks up new ones without a restart; if one does not show, run <code>/reload-skills</code>.</p>
            <Stack v-for="c in commands" :key="c.title" :gap="2">
              <h3>{{ c.title }}</h3>
              <p>{{ c.text }}</p>
              <Island when="idle" :component="CopyCommand" :props="{ command: c.command }" />
            </Stack>
          </Section>
          <Section title="Files for agents">
            <ul>
              <li><Link to="/index.md">index.md</Link>: the brief to start from.</li>
              <li><Link to="/llms.txt">llms.txt</Link>: a line on every doc, component and skill.</li>
              <li><Link to="/llms-full.txt">llms-full.txt</Link>: the docs and skills in full.</li>
              <li><Link to="/catalog.json">catalog.json</Link>: every component, and each one alone, such as <Link to="/components/date-picker.json">/components/date-picker.json</Link>.</li>
            </ul>
          </Section>
        </div>
      </Page>
    </div>
    <SiteFooter />
  </div>
</template>

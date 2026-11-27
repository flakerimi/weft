// A page is hydrated only inside its islands, so the build drops every
// comment outside them: Vue's hydration markers (<!--[-->, <!--[!-->,
// <!--[$-->, <!--[1-->, <!--]-->, <!---->, …, whatever a Vue build writes) and a
// template's own comments, bytes that do nothing there. Inside a
// <weft-island> everything stays, since the island hydrates against it.
import { expect, it } from 'vitest'
import { dropStaticComments } from '../src/build.ts'

it('drops comments outside islands and keeps everything inside them', () => {
  const island = '<weft-island data-id="Copy" data-when="idle"><!--[--><button>Copy</button><!--]--><!----><script type="application/json">{}</script></weft-island>'
  const html = `<div class="site"><!--[--><p>One</p><!-- a note --><!---->${island}<section><!--[!-->Two<!--]--><!--[$--><i>3</i><b>4</b><!--]--></section><!--]--></div>`
  expect(dropStaticComments(html)).toBe(`<div class="site"><p>One</p>${island}<section>Two<i>3</i><b>4</b></section></div>`)
})

it('keeps the comments of every island, one after another or nested', () => {
  const html = '<!--[--><weft-island data-id="A"><!--[-->a<weft-island data-id="B"><!--b--></weft-island><!--]--></weft-island><!--x--><weft-island data-id="C"><!--c--></weft-island>'
  expect(dropStaticComments(html)).toBe('<weft-island data-id="A"><!--[-->a<weft-island data-id="B"><!--b--></weft-island><!--]--></weft-island><weft-island data-id="C"><!--c--></weft-island>')
})

it('leaves a page with no comments as it is', () => {
  expect(dropStaticComments('<main><h1>Hi</h1></main>')).toBe('<main><h1>Hi</h1></main>')
})

it('leaves the text of raw-text elements as it is, outside islands too', () => {
  const html = [
    '<!--[-->',
    '<script>if (a<!--b) x()</script>',
    '<style>/* <!-- */ p { color: red }</style>',
    '<textarea><!-- typed --></textarea>',
    '<title>A <!-- title --></title>',
    '<p>x</p><!--]-->',
  ].join('')
  expect(dropStaticComments(html)).toBe(
    '<script>if (a<!--b) x()</script><style>/* <!-- */ p { color: red }</style><textarea><!-- typed --></textarea><title>A <!-- title --></title><p>x</p>',
  )
})

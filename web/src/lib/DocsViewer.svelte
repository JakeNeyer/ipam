<script>
  import { marked } from 'marked'
  import { markedHighlight } from 'marked-highlight'
  import hljs from 'highlight.js/lib/core'
  import json from 'highlight.js/lib/languages/json'
  import DOMPurify from 'dompurify'
  import { afterUpdate } from 'svelte'

  /** Clean bash grammar — comments, strings, variables only. */
  function bashLight(hljs) {
    return {
      name: 'Bash',
      aliases: ['sh'],
      contains: [
        hljs.HASH_COMMENT_MODE,
        { className: 'string', begin: /"/, end: /"/, contains: [
          { className: 'variable', begin: /\$[\w]+/ },
          { className: 'variable', begin: /\$\{/, end: /\}/ },
        ]},
        { className: 'string', begin: /'/, end: /'/ },
        { className: 'variable', begin: /\$[\w]+/ },
        { className: 'variable', begin: /\$\{/, end: /\}/ },
      ],
    }
  }

  /** Minimal HCL / Terraform grammar. */
  function hcl(hljs) {
    return {
      name: 'HCL',
      aliases: ['terraform', 'tf'],
      keywords: {
        keyword: 'resource data provider variable output locals module terraform required_providers',
        literal: 'true false null',
      },
      contains: [
        hljs.HASH_COMMENT_MODE,
        hljs.C_LINE_COMMENT_MODE,
        hljs.QUOTE_STRING_MODE,
        hljs.NUMBER_MODE,
        { className: 'section',
          begin: /\b(resource|data|provider|module|variable|output|locals|terraform)\b/,
          end: /\{/, excludeEnd: true,
          contains: [hljs.QUOTE_STRING_MODE],
        },
        { className: 'variable',
          begin: /\b[a-zA-Z_][\w]*\.[a-zA-Z_][\w]*(?:\.[a-zA-Z_][\w]*)*/,
        },
        { className: 'attr',
          begin: /^\s*[a-zA-Z_][\w]*/,
          end: /\s*=/, excludeEnd: true,
        },
      ],
    }
  }

  // Register languages
  hljs.registerLanguage('bash', bashLight)
  hljs.registerLanguage('json', json)
  hljs.registerLanguage('hcl', hcl)

  // Configure marked with syntax highlighting
  marked.use(
    markedHighlight({
      highlight(code, lang) {
        if (lang && hljs.getLanguage(lang)) {
          return hljs.highlight(code, { language: lang }).value
        }
        return code
      },
    }),
  )

  /** @type {string} Raw markdown content to render. */
  export let content = ''

  /** @type {HTMLDivElement} */
  let container

  $: rawHtml = content ? marked.parse(content, { gfm: true }) : ''
  $: html = rawHtml
    ? DOMPurify.sanitize(rawHtml, {
        ALLOWED_TAGS: [
          'p', 'br', 'strong', 'em', 'u', 's', 'a',
          'ul', 'ol', 'li', 'h1', 'h2', 'h3', 'h4',
          'code', 'pre', 'span', 'img',
          'dl', 'dt', 'dd', 'blockquote', 'hr',
        ],
        ALLOWED_ATTR: ['href', 'src', 'alt', 'class'],
      })
    : ''

  const COPY_ICON = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><rect x="9" y="9" width="13" height="13" rx="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>`
  const CHECK_ICON = `<svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"/></svg>`

  afterUpdate(() => {
    if (!container) return

    // Remove orphaned wrappers left behind when {@html} replaces content
    container.querySelectorAll('.code-block').forEach((block) => {
      if (!block.querySelector('pre')) block.remove()
    })

    container.querySelectorAll('pre:not(.code-wrapped)').forEach((pre) => {
      pre.classList.add('code-wrapped')

      const code = pre.querySelector('code')
      if (!code) return

      // Build wrapper
      const wrapper = document.createElement('div')
      wrapper.className = 'code-block'

      // Copy button (appears on hover)
      const copyBtn = document.createElement('button')
      copyBtn.className = 'code-copy-btn'
      copyBtn.type = 'button'
      copyBtn.title = 'Copy to clipboard'
      copyBtn.innerHTML = COPY_ICON
      copyBtn.addEventListener('click', () => {
        navigator.clipboard.writeText(code.textContent || '').then(() => {
          copyBtn.innerHTML = CHECK_ICON
          copyBtn.classList.add('copied')
          setTimeout(() => {
            copyBtn.innerHTML = COPY_ICON
            copyBtn.classList.remove('copied')
          }, 2000)
        })
      })

      pre.parentNode.insertBefore(wrapper, pre)
      wrapper.appendChild(pre)
      wrapper.appendChild(copyBtn)
    })
  })
</script>

<div class="docs-viewer" class:empty={!content} bind:this={container}>
  {#if content}
    {@html html}
  {:else}
    <p>Select a topic from the sidebar.</p>
  {/if}
</div>

<style>
  .docs-viewer {
    flex: 1;
    min-width: 0;
  }
  .docs-viewer :global(h1) {
    margin: 0 0 1rem 0;
    font-size: 1.75rem;
    font-weight: 600;
  }
  .docs-viewer :global(h2) {
    margin: 1.5rem 0 0.5rem 0;
    font-size: 1.1rem;
    font-weight: 600;
  }
  .docs-viewer :global(p) {
    margin: 0 0 0.75rem 0;
    line-height: 1.6;
  }
  .docs-viewer :global(ul),
  .docs-viewer :global(ol) {
    margin: 0 0 0.75rem 0;
    padding-left: 1.5rem;
  }
  .docs-viewer :global(li) {
    margin-bottom: 0.25rem;
  }
  /* Inline code */
  .docs-viewer :global(code) {
    padding: 0.1rem 0.35rem;
    font-family: var(--font-mono);
    font-size: 0.9em;
    background: var(--surface-elevated);
    border: 1px solid var(--border);
    border-radius: 3px;
  }
  .docs-viewer :global(a) {
    color: var(--accent);
    text-decoration: none;
  }
  .docs-viewer :global(a:hover) {
    text-decoration: underline;
  }
  .docs-viewer :global(img) {
    max-width: 100%;
    height: auto;
    display: block;
    margin: 1rem 0;
    border-radius: var(--radius);
    border: 1px solid var(--border);
  }
  .docs-viewer :global(.docs-data-model) {
    border-radius: 12px;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.06);
  }
  .docs-viewer :global(.screenshot-light) {
    display: block;
  }
  .docs-viewer :global(.screenshot-dark) {
    display: none;
  }
  :global(html.dark) .docs-viewer :global(.screenshot-light) {
    display: none;
  }
  :global(html.dark) .docs-viewer :global(.screenshot-dark) {
    display: block;
  }

  /* ── Code blocks ── */
  .docs-viewer :global(.code-block) {
    position: relative;
    margin: 0.75rem 0 1rem;
    border-radius: 10px;
    border: 1px solid var(--border);
    background: var(--surface);
    overflow: hidden;
  }
  /* Reset inline-code styles inside pre */
  .docs-viewer :global(pre code) {
    padding: 0;
    background: none;
    border: none;
    border-radius: 0;
    font-size: 0.85rem;
    color: var(--text);
    line-height: 1.7;
  }
  .docs-viewer :global(pre.code-wrapped) {
    margin: 0;
    padding: 0.875rem 2.5rem 0.875rem 1rem;
    background: transparent;
    border: none;
    overflow-x: auto;
  }
  /* Copy button */
  .docs-viewer :global(.code-copy-btn) {
    position: absolute;
    top: 0.5rem;
    right: 0.5rem;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 5px;
    border: 1px solid transparent;
    border-radius: 6px;
    background: transparent;
    color: var(--text-muted);
    cursor: pointer;
    opacity: 0;
    transition: opacity 0.15s, color 0.15s, background 0.15s, border-color 0.15s;
  }
  .docs-viewer :global(.code-block:hover .code-copy-btn) {
    opacity: 1;
  }
  .docs-viewer :global(.code-copy-btn:hover) {
    color: var(--text);
    background: var(--surface-elevated);
    border-color: var(--border);
  }
  .docs-viewer :global(.code-copy-btn.copied) {
    opacity: 1;
    color: #16a34a;
  }
  /* Scrollbar */
  .docs-viewer :global(pre.code-wrapped::-webkit-scrollbar) {
    height: 6px;
  }
  .docs-viewer :global(pre.code-wrapped::-webkit-scrollbar-thumb) {
    background: var(--border);
    border-radius: 3px;
  }
  .docs-viewer :global(pre.code-wrapped::-webkit-scrollbar-track) {
    background: transparent;
  }

  /* ── Syntax highlighting — light mode ── */
  .docs-viewer :global(.hljs-comment),
  .docs-viewer :global(.hljs-meta) {
    color: #6a737d;
  }
  .docs-viewer :global(.hljs-keyword),
  .docs-viewer :global(.hljs-selector-tag),
  .docs-viewer :global(.hljs-type) {
    color: #d73a49;
  }
  .docs-viewer :global(.hljs-string),
  .docs-viewer :global(.hljs-addition) {
    color: #22863a;
  }
  .docs-viewer :global(.hljs-number),
  .docs-viewer :global(.hljs-literal) {
    color: #005cc5;
  }
  .docs-viewer :global(.hljs-built_in),
  .docs-viewer :global(.hljs-title),
  .docs-viewer :global(.hljs-title.function_) {
    color: #6f42c1;
  }
  .docs-viewer :global(.hljs-variable),
  .docs-viewer :global(.hljs-template-variable) {
    color: #e36209;
  }
  .docs-viewer :global(.hljs-attr),
  .docs-viewer :global(.hljs-attribute) {
    color: #005cc5;
  }
  .docs-viewer :global(.hljs-section) {
    color: #005cc5;
    font-weight: 600;
  }
  .docs-viewer :global(.hljs-symbol),
  .docs-viewer :global(.hljs-bullet) {
    color: #005cc5;
  }
  .docs-viewer :global(.hljs-subst) {
    color: #24292e;
  }
  .docs-viewer :global(.hljs-deletion) {
    color: #b31d28;
  }

  /* ── Syntax highlighting — dark mode ── */
  :global(html.dark) .docs-viewer :global(.hljs-comment),
  :global(html.dark) .docs-viewer :global(.hljs-meta) {
    color: #8b949e;
  }
  :global(html.dark) .docs-viewer :global(.hljs-keyword),
  :global(html.dark) .docs-viewer :global(.hljs-selector-tag),
  :global(html.dark) .docs-viewer :global(.hljs-type) {
    color: #ff7b72;
  }
  :global(html.dark) .docs-viewer :global(.hljs-string),
  :global(html.dark) .docs-viewer :global(.hljs-addition) {
    color: #7ee787;
  }
  :global(html.dark) .docs-viewer :global(.hljs-number),
  :global(html.dark) .docs-viewer :global(.hljs-literal) {
    color: #79c0ff;
  }
  :global(html.dark) .docs-viewer :global(.hljs-built_in),
  :global(html.dark) .docs-viewer :global(.hljs-title),
  :global(html.dark) .docs-viewer :global(.hljs-title.function_) {
    color: #d2a8ff;
  }
  :global(html.dark) .docs-viewer :global(.hljs-variable),
  :global(html.dark) .docs-viewer :global(.hljs-template-variable) {
    color: #ffa657;
  }
  :global(html.dark) .docs-viewer :global(.hljs-attr),
  :global(html.dark) .docs-viewer :global(.hljs-attribute) {
    color: #79c0ff;
  }
  :global(html.dark) .docs-viewer :global(.hljs-section) {
    color: #79c0ff;
    font-weight: 600;
  }
  :global(html.dark) .docs-viewer :global(.hljs-symbol),
  :global(html.dark) .docs-viewer :global(.hljs-bullet) {
    color: #79c0ff;
  }
  :global(html.dark) .docs-viewer :global(.hljs-subst) {
    color: #c9d1d9;
  }
  :global(html.dark) .docs-viewer :global(.hljs-deletion) {
    color: #ffa198;
  }
</style>

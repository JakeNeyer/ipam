<script>
  import { marked } from 'marked'

  /** @type {string} Raw markdown content to render. */
  export let content = ''

  $: html = content ? marked.parse(content, { gfm: true }) : ''
</script>

<div class="docs-viewer" class:empty={!content}>
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
</style>

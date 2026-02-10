<script>
  /** Shared table wrapper for consistent styling across Reserved blocks, Networks, etc.
   *  Slots: header (thead content, typically one <tr> with <th>s), body (tbody content, <tr>s).
   */
</script>

<div class="table-wrap">
  <table class="table">
    <thead>
      <slot name="header"></slot>
    </thead>
    <tbody>
      <slot name="body"></slot>
    </tbody>
  </table>
</div>

<style>
  .table-wrap {
    overflow-x: auto;
    background: var(--surface);
    border: 1px solid var(--border);
    border-radius: var(--radius);
    box-shadow: var(--shadow-sm);
  }
  .table {
    width: 100%;
    min-width: max-content;
    border-collapse: collapse;
    font-size: 0.9rem;
  }
  /* Slotted content (header/body from parent) - use :global so styles apply to slot output */
  .table-wrap :global(.table th) {
    text-align: left;
    padding: 0.75rem 1rem;
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    color: var(--text-muted);
    background: var(--table-header-bg, var(--bg));
    border-bottom: 1px solid var(--border);
  }
  .table-wrap :global(.table th.sortable) {
    padding: 0;
  }
  .table-wrap :global(th .th-sort) {
    display: inline-flex;
    flex-direction: row;
    flex-wrap: nowrap;
    align-items: center;
    justify-content: space-between;
    gap: 0.35rem;
    width: 100%;
    min-width: 0;
    padding: 0.75rem 1rem;
    text-align: left;
    white-space: nowrap;
    font-size: inherit;
    font-weight: inherit;
    text-transform: inherit;
    letter-spacing: inherit;
    color: inherit;
    background: none;
    border: none;
    cursor: pointer;
    font-family: inherit;
    transition: color 0.15s, background 0.15s;
  }
  .table-wrap :global(th .th-sort:hover) {
    color: var(--text);
    background: rgba(255, 255, 255, 0.04);
  }
  .table-wrap :global(th.sortable.sorted .th-sort) {
    color: var(--accent);
  }
  .table-wrap :global(th .th-sort-label) {
    flex-shrink: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .table-wrap :global(th .sort-icon) {
    flex-shrink: 0;
    flex-grow: 0;
    font-size: 0.65rem;
    opacity: 0.9;
    display: inline-flex;
    align-items: center;
    line-height: 0;
    vertical-align: middle;
  }
  .table-wrap :global(th .sort-icon svg) {
    display: block;
    vertical-align: middle;
  }
  .table-wrap :global(.table td) {
    padding: 0.75rem 1rem;
    border-bottom: 1px solid var(--table-row-border, var(--border));
    vertical-align: middle;
  }
  .table-wrap :global(.table tbody tr:last-child td) {
    border-bottom: none;
  }
  .table-wrap :global(.table tbody tr:hover td) {
    background: var(--table-row-hover, rgba(255, 255, 255, 0.02));
  }
  .table-wrap :global(.table .actions),
  .table-wrap :global(.table .table-actions) {
    text-align: right;
    white-space: nowrap;
    min-width: 120px;
  }
  .table-wrap :global(.table thead th:first-child) {
    min-width: 160px;
  }
  .table-wrap :global(.table td.name) {
    min-width: 160px;
  }
  .table-wrap :global(.table td.cidr-cell code),
  .table-wrap :global(.table td.cidr code) {
    font-size: 0.85em;
    background: var(--table-header-bg, var(--bg));
    padding: 0.15em 0.4em;
    border-radius: 3px;
    color: var(--text);
  }
</style>

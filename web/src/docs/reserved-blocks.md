# Reserved blocks

Reserved blocks are CIDR ranges that are set aside and cannot be used for normal network blocks or allocations. They are managed on this page (admin only) and appear on the Networks page so everyone can see what is reserved. The app rejects any attempt to create a block or allocation that overlaps a reserved range. Use reserved blocks for future use, DMZ space, or other ranges you want to keep off-limits to normal allocation.

<p class="docs-screenshot">
<img src="/images/reserved-blocks-light.png" alt="Reserved blocks page (light mode)" class="screenshot-light" />
<img src="/images/reserved-blocks-dark.png" alt="Reserved blocks page (dark mode)" class="screenshot-dark" />
</p>

## Managing reserved blocks

Add a reserved range by providing a name, CIDR, and reason (e.g. "Future use", "DMZ"). Once added, that range is blocked: no new block or allocation can overlap it. This prevents accidental use of space you have designated for something else. Reserved blocks are visible on the Networks page and in the Dashboard resource graph under "Reserved", so the whole team can see what is reserved. Edit or delete reserved blocks from this page when your plans change.

## Who can access Reserved blocks

Only users with the admin role can open the Reserved blocks page and add, edit, or remove reserved ranges. Other users can see reserved ranges on Networks and the Dashboard but cannot change them.

# CIDR wizard

The CIDR wizard suggests non-overlapping CIDR ranges when creating network blocks or allocations. It appears on the Networks page when you click **Create block** or **Create allocation**.

<p class="docs-screenshot">
<img src="/images/cidr-wizard-light.png" alt="CIDR wizard (light mode)" class="screenshot-light" />
<img src="/images/cidr-wizard-dark.png" alt="CIDR wizard (dark mode)" class="screenshot-dark" />
</p>

- **For blocks** — Set a base address (e.g. `10.0.0.0`) and prefix length (e.g. `/16`), or use a suggested CIDR that avoids existing blocks and reserved ranges.
- **For allocations** — Select a parent block, choose a prefix length (e.g. `/24` for 256 IPs), and the wizard suggests the next available subnet that fits without overlaps.
- **Manual entry** — You can always type a CIDR manually instead of using a suggestion.

The wizard shows IP counts for each prefix length (e.g. `/24` = 256, `/16` = 65,536) and prevents overlapping ranges.

# Network Advisor

The Network Advisor is a step-by-step wizard that starts from a **base CIDR**, then defines **environments (each with at least one pool)**, optional **reserved blocks**, and **network blocks**. It recommends suggested network layouts based on common topologies and sizes.

<p class="docs-screenshot">
<img src="/images/network-advisor-light.png" alt="Network Advisor (light mode)" class="screenshot-light" />
<img src="/images/network-advisor-dark.png" alt="Network Advisor (dark mode)" class="screenshot-dark" />
</p>

## Steps

1. **Base CIDR** — Choose the overall address space (e.g. `10.0.0.0/8`). Environments, their pools, and network blocks are carved from this range.
2. **Environments & pools** — Define environments from a template (`SDLC`, `Cloud`, `Hybrid`) or create your own. Each environment gets at least one pool (a subnet of the base CIDR); you can add more pools per environment later on the Networks page.
3. **Reserved blocks** — Optionally reserve CIDR ranges within the base; they are carved out before environment pools and blocks.
4. **Network blocks** — Set the number of network blocks per environment and hosts per network. Blocks are created inside each environment’s pool. A progress bar shows usage of the base range.
5. **Summary** — Review the plan and click **Generate resources from plan** to create environments (each with its pool), network blocks, and optionally reserved blocks.

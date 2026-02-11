# Network Advisor

The Network Advisor is a step-by-step wizard for planning your IP strategy and generating environments, blocks, and reserved ranges in one flow. It recommends suggested network layouts based on common topologies and sizes.

<p class="docs-screenshot">
<img src="/images/network-advisor-light.png" alt="Network Advisor (light mode)" class="screenshot-light" />
<img src="/images/network-advisor-dark.png" alt="Network Advisor (dark mode)" class="screenshot-dark" />
</p>

## Steps

1. **Base CIDR** — Choose a starting range (e.g. `10.0.0.0/8`) or enter a custom CIDR.
2. **Environments** — Define environments from a template (`SDLC`, `Cloud`, `Hybrid`) or create your own.
3. **Reserved blocks** — Optionally reserve CIDR ranges that should not be allocated to environments.
4. **Block sizing** — Set the number of networks per environment with a slider or typed input. Hosts per network and total IPs are calculated automatically. A progress bar shows aggregate capacity usage.
5. **Summary** — Review the plan and click **Generate resources from plan** to create everything at once.

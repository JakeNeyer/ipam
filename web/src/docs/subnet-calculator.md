# Subnet calculator

The Subnet calculator helps you explore IPv4 subnets without creating IPAM resources. Enter a network address and prefix length to see the resulting subnet, then use **Divide** to split any row into two equal subnets or **Join** to merge two halves back into one.

<p class="docs-screenshot">
<img src="/images/subnet-calculator-light.png" alt="Subnet calculator (light mode)" class="screenshot-light" />
<img src="/images/subnet-calculator-dark.png" alt="Subnet calculator (dark mode)" class="screenshot-dark" />
</p>

## How to use

1. Enter a **network address** (e.g. `10.0.0.0`) and choose a **mask** (e.g. `/24`). The table shows the subnet with its address range, usable IPs, and host count.
2. Click **Divide** on any row to split that subnet into two equal subnets (e.g. `/24` → two `/25` subnets). You can keep dividing rows to plan smaller subnets.
3. When two adjacent rows are halves of the same parent subnet, **Join** appears. Click it to merge them back into a single row.

Changing the network or mask at the top resets the table to a single subnet so you can start a new calculation. The calculator does not create or modify IPAM environments, blocks, or allocations—it is for planning and reference only.

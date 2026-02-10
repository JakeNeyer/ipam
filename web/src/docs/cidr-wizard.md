# CIDR wizard

The CIDR wizard helps you choose a CIDR range when creating a network block or allocation. It appears on the Networks page when you click **Create block** or **Create allocation**. It suggests ranges that do not overlap existing blocks or allocations, so you can avoid conflicts and pick the right size.

<p class="docs-screenshot">
<img src="/images/cidr-wizard-light.png" alt="CIDR wizard (light mode)" class="screenshot-light" />
<img src="/images/cidr-wizard-dark.png" alt="CIDR wizard (dark mode)" class="screenshot-dark" />
</p>

## When creating a block

When you create a network block, the wizard lets you:

- **Set the base address** — Enter the four IPv4 octets (e.g. 10, 0, 0, 0) that start the range.
- **Choose the prefix length** — Select a prefix (e.g. /16, /24) to set the block size. The wizard shows how many IPs that gives (e.g. 65,536 for /16).
- **Use a suggested CIDR** — If you have selected an environment, the wizard can suggest a CIDR that does not overlap any existing blocks or reserved ranges in that environment. Click **Use this CIDR** to fill the CIDR field with the suggestion.
- **Use a manual CIDR** — If you are not using a suggestion, set the base address and prefix, then click **Use manual CIDR** to fill the CIDR field with the resulting range (e.g. 10.0.0.0/16).

The suggested CIDR is especially useful when you have many blocks already; it finds a gap so you do not have to guess.

## When creating an allocation

When you create an allocation, you first select a block. The wizard then:

- **Shows the parent block** — Displays the block’s CIDR so you know which range you are subdividing.
- **Lets you choose the subnet size** — Select a prefix length for the allocation (e.g. /24 for 256 IPs). The prefix must be smaller than the block’s (e.g. /24 inside a /16 block).
- **Suggests the next available subnet** — The wizard suggests a CIDR that fits inside the block and does not overlap existing allocations in that block. It uses bin-packing so the suggestion fills gaps when possible. Click **Use this CIDR** to fill the CIDR field.

You can still type a CIDR manually in the form; the wizard is there to help you pick a valid, non-overlapping range quickly.

## Why use the wizard

- **Avoid overlaps** — Suggestions respect existing blocks, allocations, and reserved ranges so you do not create conflicting ranges.
- **See IP counts** — Each prefix length shows how many IPs you get (e.g. /24 = 256, /16 = 65,536).
- **See the range** — For suggested or manual CIDRs, the wizard can show the first and last IP in the range so you can confirm it is what you want.

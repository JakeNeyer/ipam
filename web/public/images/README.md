# User guide screenshots

Screenshots for the in-app user guide (light and dark mode) live here. The script **logs in first** so all pages show authenticated content. To capture or refresh them:

1. Start the app: `npm run dev` (from `web/`).
2. (First time only) Install Chromium: `npx playwright install chromium`
3. Run with login credentials: `LOGIN_EMAIL=admin@localhost LOGIN_PASSWORD=yourpassword npm run screenshot-docs`

Without `LOGIN_EMAIL` and `LOGIN_PASSWORD`, the script skips login and screenshots may show the login page.

To regenerate placeholder PNGs only: `npm run create-placeholder-screenshots`.

Files:

- `dashboard-light.png` / `dashboard-dark.png` — Dashboard (`#`)
- `networks-light.png` / `networks-dark.png` — Networks (`#networks`)
- `environments-light.png` / `environments-dark.png` — Environments (`#environments`)
- `admin-light.png` / `admin-dark.png` — Admin (`#admin`)
- `reserved-blocks-light.png` / `reserved-blocks-dark.png` — Reserved blocks (`#reserved-blocks`)
- `command-palette-light.png` / `command-palette-dark.png` — Command palette (⌘K)
- `cidr-wizard-light.png` / `cidr-wizard-dark.png` — CIDR wizard (Create block)
- `subnet-calculator-light.png` / `subnet-calculator-dark.png` — Subnet calculator (`#subnet-calculator`)
- `logo.svg` / `logo-light.svg` — App logo (used in Nav, Landing, Login, Setup)

The docs viewer shows the light screenshot in light mode and the dark screenshot in dark mode.

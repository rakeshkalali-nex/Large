# Task 2 — architectural theme evidence

## Scope and integration

Theme-only implementation against design.md lines 415–443 and task 2. Entry point: `frontend/src/styles.scss`; partials: `_tokens.scss`, `_fonts.scss`, `_base.scss`, `_controls.scss`, `_layout.scss` under `frontend/src/styles/`. Angular build configuration is owned by the parent task; this change does not edit manifests or application components.

Global tokens preserve all specified palette values, 8px spacing rhythm, heading/body/table sizes, control/card radii and surface shadow. Native controls and CDK overlay containers use IBM Plex Sans. Utilities supply primary/secondary/destructive, disabled, pending, validation and semantic status presentation; numbers use tabular figures. Focus uses a 3px blue outline with 2px white separation; `.nav-surface` switches the ring to white. Motion is opacity-only, 150ms; reduced motion removes transitions/animations and smooth scrolling, leaving a static pending indicator with accompanying text.

Integration contracts (CSS does not implement semantics or interaction):

- Apply `.nav-surface` to navy navigation. Use `.button--secondary` / `.button--destructive` on buttons; `.button` is the equivalent link presentation. Native `disabled` is preferred; components must suppress activation for `aria-disabled="true"`. Pending controls need visible text, `.spinner` marked decorative and `aria-busy="true"`.
- Fields need permanent associated labels, help/error IDs and `aria-describedby`. Invalid controls require `aria-invalid="true"`. Status messages require an icon plus explicit text; CSS never supplies their accessible name.
- `.form-grid`, `.form-actions`, `.page-content`, `.card`, `.dialog-surface`, `.filter-chips` and `.empty-state` are opt-in layout utilities. Dialog focus trapping, Escape, focus return, announcements, route-heading focus and named filter drawers remain component responsibilities.
- `.table-region` bounds desktop vertical scrolling with sticky headers and 48px minimum rows. `.table--stacked` opts into phone rows; use explicit table/rowgroup/row/cell roles when changing display, proper column headers and `data-label` cell text, and verify screen-reader behavior. Prefer summary cards where appropriate. Selected rows still need an actual selected checkbox/icon. Amounts use `.amount`.
- Only named `.ledger-region` or `.timeline-region` elements with `role="region"` opt into horizontal scrolling; make scrolling regions keyboard reachable. Never use body overflow clipping to conceal layout defects. Other tables must wrap or stack. Test actual content for overflow.
- 1024px reduces the sidebar-width token; 768px uses 16px page spacing and stacked full-width actions. The shell owns its mobile overlay navigation. Forms have no fixed height, dialogs use dynamic viewport bounds, safe-area bottom padding is provided, and sticky actions return to normal flow on short viewports.

## Fonts and licensing

Copied byte-for-byte from installed `@fontsource/ibm-plex-sans` **5.3.0**, not downloaded placeholders: normal 400, 500, 600 and 700 weights, with six subsets each (Latin, Latin extended, Cyrillic, Cyrillic extended, Greek, Vietnamese). All **24 WOFF2** files live in `frontend/public/fonts/ibm-plex-sans/`. The source Unicode ranges are retained, so browsers fetch only needed subsets. `font-display: swap` and system fallback avoid invisible text. Glyphs outside those ranges fall back to system fonts; italic faces are not included.

The exact upstream copyright notice and **SIL Open Font License 1.1** accompany the binaries at `frontend/public/fonts/ibm-plex-sans/LICENSE`. Copyright 2019 IBM Corp. Fonts are unmodified; do not redistribute them without this license or sell the font files by themselves. All font URLs are local `/fonts/ibm-plex-sans/...`; no remote font service is used.

Original geometric artwork: `frontend/public/branding/buildestate-mark.svg`, its shell-compatible identical alias `buildestate.svg`, and `architectural-lines.svg`. These contain no third-party art or embedded font. See `frontend/public/branding/README.md` for attribution/licensing and alt-text guidance. Render the BuildEstate wordmark as adjacent live IBM Plex Sans text; decorative lines belong on navy and must be hidden from assistive technology.

## Contrast calculations (not rendered-browser evidence)

Calculated with WCAG sRGB linearization and `(lighter luminance + 0.05) / (darker luminance + 0.05)`, rounded to two decimals:

| Foreground / background | Ratio |
| --- | ---: |
| Main text #243447 / white | 12.67:1 |
| Muted #526170 / white | 6.36:1 |
| White / navigation #14283F | 14.95:1 |
| White / primary #0F766E | 5.47:1 |
| White / hover #115E59 | 7.58:1 |
| White / pressed #134E4A | 9.48:1 |
| Success #166534 / #DCFCE7 | 6.49:1 |
| Error #B91C1C / #FEF2F2 | 5.91:1 |
| Warning #B45309 / #FFF7ED | 4.73:1 |
| Information #1D4ED8 / #EFF6FF | 6.16:1 |
| Disabled text #526170 / #E2E8F0 | 5.16:1 |
| Focus blue #1D4ED8 / white | 6.70:1 |
| Border #CBD5E1 / white | **1.48:1** |

Specified text pairs meet 4.5:1. The specified pale border does **not** reach 3:1 and must not alone identify interactive controls: text fields add a muted-slate inset edge; secondary buttons use a muted-slate edge. Decorative card/table separators keep the exact border token. Status text/icons and selected checkboxes must convey meaning independently of color. This is not a blanket WCAG compliance claim.

## Verification completed

On 2026-11-03, a direct in-memory Sass compile succeeded (19,772 bytes CSS). A Node assertion check verified all 24 files have the WOFF2 signature, exactly match installed source bytes, and appear in compiled local CSS URLs; the copied license exactly matches upstream. The same command calculated the contrast ratios above. No application build, unit suite or browser test was run by this theme task; concurrent component work belongs to other tasks.

## Pending real-browser evidence (task 30)

### Shared pending-button contrast correction

The pending-button component scopes hover and pressed backgrounds to each enabled variant, rather than applying teal hover to secondary and destructive actions. Secondary keeps slate `#243447` text on white, warm `#F5F6F3` hover and `#E2E8F0` pressed backgrounds. Its `#526170` border also supplies an identifiable control edge. Destructive keeps white text on red `#B91C1C`, darkening to `#991B1B` on hover and `#7F1D1D` when pressed. Primary uses the documented teal palette. Pressed states retain a current-color inset edge without movement; disabled/pending states retain the readable neutral palette and native disabled behavior.

`frontend/src/app/shared/pending-button.spec.ts` checks emitted variant-specific enabled-state rules and calculates WCAG text contrast (at least 4.5:1), plus the secondary default border (at least 3:1), and checks disabled/pending activation for every variant. These are unit-level palette checks, not proof of real-browser pseudo-class rendering; that remains in the browser checks below.

- Confirm actual served fonts in network requests (local origin, successful responses, no external font calls) and rendered/computed fonts for 400/500/600/700 on body, buttons, inputs, tables and opened CDK overlays. Confirm fallback/swap under delayed or failed loading.
- Exercise keyboard navigation, visible unclipped rings on white/nav/scroll regions, dialog trap/Escape/return, status announcements, disabled/pending activation, invalid summaries and route focus.
- Inspect desktop, 1024px, 768px and 390px layouts, 200%/400% zoom, long labels and amounts, stacked table screen-reader header relationships and named ledger/timeline keyboard scrolling. Confirm no page-level horizontal scrolling.
- Check real phone software-keyboard behavior, dynamic viewport and safe-area padding; ensure Sign in and form actions remain reachable by document scrolling.
- Emulate reduced motion and forced colors, and check that pending text remains meaningful without animation. Verify final rendered contrast including inherited/component overrides.
- Record current/previous supported Chrome, Edge, Firefox and Safari versions plus physical mobile-device checks. None of these browser/device checks is claimed passed by this document.

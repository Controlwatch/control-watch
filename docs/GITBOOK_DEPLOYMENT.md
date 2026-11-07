# GitBook deployment

This repository is GitBook-ready: `README.md` is the landing page,
`SUMMARY.md` defines navigation, and `.gitbook.yaml` points GitBook at both.

To publish it, create a GitBook space and connect the GitHub repository
`Controlwatch/control-watch` using Git Sync. Select the `main` branch and the
repository root, then run a sync. GitBook will use `SUMMARY.md` as the page
tree; subsequent pushes to `main` can be synced from the connected space.

No GitBook access token is stored in this repository. Deployment therefore must
be completed by an authenticated repository owner in the GitBook workspace.
After the space URL exists, set that URL as the GitHub repository homepage.

Source of truth: `.gitbook.yaml`, `SUMMARY.md`, and GitBook's connected-space
settings.

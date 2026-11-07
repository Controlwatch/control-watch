# GitBook deployment

This repository is GitBook-ready: `docs/README.md` is the landing page,
`docs/SUMMARY.md` defines navigation, and `gitbook-docs.yaml` maps the
documentation space to `./docs`.

To publish it, create a GitBook space and connect the GitHub repository
`Controlwatch/control-watch` using Git Sync. Select the `main` branch and the
repository root (`./`), then run a GitHub → GitBook sync. GitBook will read
`gitbook-docs.yaml`, map the ControlWatch space to `./docs`, and use
`docs/SUMMARY.md` as the page tree.

No GitBook access token is stored in this repository. Deployment therefore must
be completed by an authenticated repository owner in the GitBook workspace.
After the space URL exists, set that URL as the GitHub repository homepage.

Source of truth: `.gitbook.yaml`, `SUMMARY.md`, and GitBook's connected-space
settings.

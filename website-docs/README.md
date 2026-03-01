# Kashvi Static Docs Website

This folder contains host-ready static HTML docs.

## Files

- `index.html`
- `installation.html`
- `crud.html`
- `cli.html`
- `assets/site.css`
- `assets/site.js`

## Local Preview

From repository root:

```bash
python3 -m http.server 8081
```

Then open:

- `http://localhost:8081/website-docs/`

## Hosting

Upload the full `website-docs/` folder to any static host:

- Netlify
- Vercel static
- GitHub Pages
- S3 + CloudFront
- Nginx/Apache static root

Use `website-docs/index.html` as the homepage.

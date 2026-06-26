// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import starlightLlmsTxt from "starlight-llms-txt";

// Apex (root) domain — no Pages `base`/rehype base-link plugin needed.
const SITE = "https://uficli.sh";

export default defineConfig({
  site: SITE,
  integrations: [
    starlight({
      title: "ufi",
      description:
        "An agent-friendly CLI for Ubiquiti UniFi Network, built on Ubiquiti's official local Integration API. Read-only by default; mutations gated; config reviewed by hash; bounded JSON; injection-fenced.",
      logo: { src: "./src/assets/mark.svg", alt: "ufi" },
      customCss: ["./src/styles/tokens.css", "./src/styles/docs.css"],
      social: [
        { icon: "github", label: "GitHub", href: "https://github.com/rnwolfe/ufi" },
      ],
      plugins: [starlightLlmsTxt()],
      // Head override sets per-page og:image/twitter:image (see src/components/Head.astro).
      components: { Head: "./src/components/Head.astro" },
      head: [
        { tag: "meta", attrs: { property: "og:type", content: "website" } },
        { tag: "meta", attrs: { name: "twitter:card", content: "summary_large_image" } },
        {
          tag: "link",
          attrs: {
            rel: "stylesheet",
            href: "https://fonts.googleapis.com/css2?family=Inter+Tight:wght@400;500;600;700&family=JetBrains+Mono:wght@400;500;700;800&display=swap",
          },
        },
      ],
      sidebar: [
        { label: "Start", items: [{ slug: "getting-started" }, { slug: "auth" }] },
        { label: "Guides", items: [{ slug: "config" }, { slug: "agents" }] },
        { label: "Reference", items: [{ slug: "commands" }] },
      ],
      editLink: { baseUrl: "https://github.com/rnwolfe/ufi/edit/main/site/" },
    }),
  ],
});

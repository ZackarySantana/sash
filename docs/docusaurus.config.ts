import { themes as prismThemes } from "prism-react-renderer";
import type { Config } from "@docusaurus/types";
import type * as Preset from "@docusaurus/preset-classic";

/** GitHub Pages project sites need a path prefix; CI sets DOCUSAURUS_BASE_URL (e.g. `/sash/`). */
function normalizeBaseUrl(raw: string | undefined): string {
    const s = raw?.trim() || "/";
    if (s === "/") return "/";
    const lead = s.startsWith("/") ? s : `/${s}`;
    return lead.endsWith("/") ? lead : `${lead}/`;
}

const url = process.env.DOCUSAURUS_URL ?? "https://example.com";
const baseUrl = normalizeBaseUrl(process.env.DOCUSAURUS_BASE_URL);

const config: Config = {
    title: "Sash",
    tagline:
        "Ship a Go-powered browser UI without rewriting the same glue—one config file, typed RPC, optional SSE.",
    favicon: "img/favicon.ico",

    future: {
        v4: true,
    },

    url,
    baseUrl,

    organizationName: "zackarysantana",
    projectName: "sash",

    onBrokenLinks: "throw",

    i18n: {
        defaultLocale: "en",
        locales: ["en"],
    },

    presets: [
        [
            "classic",
            {
                docs: {
                    sidebarPath: "./sidebars.ts",
                    editUrl:
                        "https://github.com/zackarysantana/sash/edit/main/docs/docs/",
                },
                blog: false,
                theme: {
                    customCss: "./src/css/custom.css",
                },
            } satisfies Preset.Options,
        ],
    ],

    themeConfig: {
        image: "img/docusaurus-social-card.jpg",
        colorMode: {
            respectPrefersColorScheme: true,
        },
        navbar: {
            title: "Sash",
            logo: {
                alt: "Sash",
                src: "img/logo.svg",
            },
            items: [
                {
                    type: "docSidebar",
                    sidebarId: "docsSidebar",
                    position: "left",
                    label: "Documentation",
                },
                {
                    href: "https://github.com/zackarysantana/sash",
                    label: "GitHub",
                    position: "right",
                },
            ],
        },
        footer: {
            style: "dark",
            links: [
                {
                    title: "Docs",
                    items: [
                        {
                            label: "Introduction",
                            to: "/docs/intro",
                        },
                        {
                            label: "Getting started",
                            to: "/docs/getting-started",
                        },
                    ],
                },
                {
                    title: "Repository",
                    items: [
                        {
                            label: "GitHub",
                            href: "https://github.com/zackarysantana/sash",
                        },
                    ],
                },
            ],
            copyright: `Copyright © ${new Date().getFullYear()} Zackary Santana. Built with Docusaurus.`,
        },
        prism: {
            theme: prismThemes.github,
            darkTheme: prismThemes.dracula,
        },
    } satisfies Preset.ThemeConfig,
};

export default config;

import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'WooCoo',
  tagline: 'WooCoo是一套实践性强,为低代码而努力Web及RPC的开发框架',
  url: 'https://tsingsun.github.io',
  baseUrl: '/woocoo',
  onBrokenLinks: 'throw',
  onBrokenMarkdownLinks: 'warn',
  favicon: 'img/favicon.ico',

  // GitHub pages deployment config.
  // If you aren't using GitHub pages, you don't need these.
  organizationName: 'woocoo', // Usually your GitHub org/user name.
  projectName: 'woocoo', // Usually your repo name.

  // Even if you don't use internalization, you can use this field to set useful
  // metadata like html lang. For example, if your site is Chinese, you may want
  // to replace "en" with "zh-Hans".
  i18n: {
    defaultLocale: 'zh',
    locales: ['zh'],
    localeConfigs: {
      cn: {
        label: '简体中文',
        direction: 'ltr',
      }
    }
  },

  presets: [
    [
      'classic',
      {
        docs: {
          path: '../md',
          sidebarPath: './sidebars.ts',
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl:
            'https://github.com/tsingsun/woocoo/tree/main/docs/md/',
        },
        blog: {
          showReadingTime: true,
          // Please change this to your repo.
          // Remove this to remove the "edit this page" links.
          editUrl:
            'https://github.com/tsingsun/woocoo/tree/main/docs/website/blog/',
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      },
    ],
  ],

  themeConfig: {
    navbar: {
      title: 'WooCoo',
      logo: {
        alt: 'WooCoo Logo',
        src: 'img/logo.svg',
      },
      items: [
        {
          type: 'doc',
          docId: 'guide',
          position: 'left',
          label: 'Docs',
        },
        {
          href: 'https://github.com/tsingsun/woocoo',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Docs',
          items: [
            {
              label: 'Get Started',
              to: '/docs/guide',
            },
          ],
        },
        {
          title: 'Community',
          items: [
            {
              label: 'Stack Overflow',
              href: 'https://stackoverflow.com/questions/tagged/woocoo',
            },
            {
              label: 'Discord',
              href: 'https://discordapp.com/invite/woocoo',
            },
            {
              label: 'Twitter',
              href: 'https://twitter.com/woocoo',
            },
          ],
        }
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Tsingsun Li. Built with Docusaurus.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
    },
  } satisfies Preset.ThemeConfig,
};

export default config;

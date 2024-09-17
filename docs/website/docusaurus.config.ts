import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';
import 'dotenv/config';

const url = process.env.DOC_URL || 'https://www.woocoos.tech',
  baseUrl = process.env.DOC_BASE_URL || '/';

const config: Config = {
  title: 'WooCoo',
  tagline: '助力开发者快速构建高性能企业级应用',
  url: url,
  baseUrl: baseUrl,
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
          feedOptions: {
            type: "all",
            copyright: `Copyright © ${new Date().getFullYear()}, The WooCoo Authors.`,
          },
          path: "blog",
          blogSidebarCount: 'ALL',
          blogSidebarTitle: 'All our posts',
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      },
    ],
  ],

  plugins: [
    ['./src/plugins/baidu-tongji/index.ts',{}]
  ],

  themeConfig: {
    announcementBar: {
      id: 'announcementBar-1',
      content:
        `⭐️ 喜欢WooCoo的话就给颗星吧 👉<a target="_blank" rel="noopener noreferrer" href="https://github.com/tsingsun/woocoo">GitHub</a>! ⭐️`,
    },
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
    algolia: {
      appId: 'F39VT0FT56',
      // YOUR_SEARCH_API_KEY
      apiKey: 'c898bb9ba001a9daee6a1b8358523985',
      indexName: 'woocoo',
      contextualSearch: true,

    },
    footer: {
      style: 'dark',
      links: [
        {
          title: '相关资源',
          items: [
            {
              label: 'Knockout(开发中)',
              href: 'https://github.com/woocoos',
            },
          ],
        },
        {
          title: '社区',
          items: [
            {
              label: 'Stack Overflow',
              href: 'https://stackoverflow.com/questions/tagged/woocoo',
            },
            {
              label: 'Discord',
              href: 'https://discord.gg/358d5uth',
            },
          ],
        }
      ],
      copyright: `Copyright © 2023 - ${new Date().getFullYear()} Tsingsun Li. <p />
<img src=${baseUrl+'img/beian.png'} style="width:16px;vertical-align: middle" alt=""/>
<a href="https://beian.mps.gov.cn/#/query/webSearch?code=35020302035958" rel="noreferrer" target="_blank">闽公网安备35020302035958</a>
<a href="https://beian.miit.gov.cn" rel="nofollow" target="_blank" style="margin-left: 20px">闽ICP备2023019801号.</a>`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      magicComments: [
        {
          className: 'theme-code-block-highlighted-line',
          line: 'highlight-next-line',
          block: {start: 'highlight-start', end: 'highlight-end'},
        },
        {
          className: 'code-block-error-message',
          line: 'highlight-next-line-error-message',
        },
        {
          className: 'code-block-info-line',
          line: 'highlight-next-line-info',
          block: {start: 'highlight-info-start', end: 'highlight-info-end'},
        },
      ],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;

import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */
const sidebars: SidebarsConfig = {
  // But you can create a sidebar manually
  md: [
    'guide',
    {
      type: 'category',
      label: '启航',
      items: ['quickstart'],
      collapsed: false,
    },
    {
      type: 'category',
      label: '乘风',
      items: ['install','conf',
        {
          type: 'category',
          label: 'Web',
          items: ['gin','graphql']
        },
        'grpc','db','cache'
      ],
      collapsed: false,
    },
    {
      type: 'category',
      label: '破浪',
      items: ['logger','otel','micro','pkg-gds'],
      collapsed: false,
    },
    {
      type: 'category',
      label: '沙滩美女',
      items: ["codegen","cli-init","oasgen","utils"],
      collapsed: false,
    }
  ],
};

module.exports = sidebars;

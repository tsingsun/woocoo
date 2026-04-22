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
  md: [
    'guide',
    {
      type: 'category',
      label: '快速开始',
      items: ['install', 'quickstart'],
      collapsed: false,
    },
    {
      type: 'category',
      label: '核心概念',
      items: ['conf', 'logger'],
      collapsed: false,
    },
    {
      type: 'category',
      label: 'Web 开发',
      items: ['gin', 'graphql'],
      collapsed: false,
    },
    {
      type: 'category',
      label: '微服务',
      items: ['grpc', 'micro'],
      collapsed: false,
    },
    {
      type: 'category',
      label: '数据与缓存',
      items: ['db', 'cache'],
      collapsed: false,
    },
    {
      type: 'category',
      label: '可观测性',
      items: ['otel'],
      collapsed: false,
    },
    {
      type: 'category',
      label: '工具链',
      items: ['cli-init', 'codegen', 'oasgen', 'utils', 'pkg-gds'],
      collapsed: false,
    }
  ],
};

export default sidebars;

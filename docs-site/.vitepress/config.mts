import { defineConfig } from 'vitepress'

export default defineConfig({
  lang: 'zh-CN',
  title: 'Recurdream 文档',
  description: 'Recurdream AI API 接入与使用指南',
  cleanUrls: true,
  lastUpdated: true,
  head: [
    ['link', { rel: 'icon', href: '/favicon.svg', type: 'image/svg+xml' }],
    ['meta', { name: 'theme-color', content: '#8b5cf6' }],
    ['meta', { property: 'og:type', content: 'website' }],
    ['meta', { property: 'og:title', content: 'Recurdream 文档' }],
    ['meta', { property: 'og:description', content: '从注册、充值、创建 API Key 到接入 Tavo、Cherry Studio、ChatBox 等客户端。' }]
  ],
  themeConfig: {
    logo: '/favicon.svg',
    siteTitle: 'Recurdream Docs',
    nav: [
      { text: '使用手册', link: '/' },
      { text: '客户端接入', link: '/clients/' },
      { text: '充值计费', link: '/billing/' }
    ],
    sidebar: [
      {
        text: '开始使用',
        items: [
          { text: '使用手册', link: '/' },
          { text: '快速开始', link: '/guide/getting-started' },
          { text: '创建 API Key', link: '/guide/api-key' },
          { text: 'API 端点与模型', link: '/guide/endpoints' }
        ]
      },
      {
        text: '客户端接入',
        items: [
          { text: '接入概览', link: '/clients/' },
          { text: 'Tavo', link: '/clients/tavo' },
          { text: 'Cherry Studio', link: '/clients/cherry-studio' },
          { text: 'ChatBox', link: '/clients/chatbox' },
          { text: 'NextChat', link: '/clients/nextchat' },
          { text: 'Claude Code', link: '/clients/claude-code' },
          { text: 'Gemini CLI', link: '/clients/gemini-cli' }
        ]
      },
      {
        text: '充值与计费',
        items: [
          { text: '充值说明', link: '/billing/' },
          { text: '余额与用量', link: '/billing/usage' }
        ]
      },
      {
        text: '帮助',
        items: [
          { text: '常见问题', link: '/faq/' }
        ]
      }
    ],
    outline: {
      level: [2, 3],
      label: '本页目录'
    },
    search: {
      provider: 'local',
      options: {
        translations: {
          button: {
            buttonText: '搜索文档',
            buttonAriaLabel: '搜索文档'
          },
          modal: {
            noResultsText: '没有找到相关内容',
            resetButtonTitle: '清除搜索',
            displayDetails: '显示详情',
            footer: {
              selectText: '选择',
              navigateText: '切换',
              closeText: '关闭'
            }
          }
        }
      }
    },
    socialLinks: [
      { icon: 'github', link: 'https://github.com/merak824/RecurdreamApi' }
    ],
    footer: {
      message: 'Recurdream 官方使用文档',
      copyright: 'Copyright © 2026 Recurdream'
    },
    editLink: {
      pattern: 'https://github.com/merak824/RecurdreamApi/edit/main/docs-site/:path',
      text: '在 GitHub 上编辑此页'
    },
    lastUpdated: {
      text: '最后更新',
      formatOptions: {
        dateStyle: 'medium',
        timeStyle: 'short'
      }
    },
    docFooter: {
      prev: '上一页',
      next: '下一页'
    },
    darkModeSwitchLabel: '切换深浅色',
    sidebarMenuLabel: '菜单',
    returnToTopLabel: '回到顶部',
    langMenuLabel: '切换语言'
  }
})

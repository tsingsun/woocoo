export default async function BaiduTongjiPlugin(context, opts) {
  return {
    name: 'baidu-tongji-plugin',
    injectHtmlTags() {
      if (process.env.NODE_ENV === 'development') {
        return {}
      }

      return {
        headTags: [
          {
            tagName: 'script',
            innerHTML: `
            var _hmt = _hmt || [];
            (function() {
              var hm = document.createElement("script");
              hm.src = "https://hm.baidu.com/hm.js?e4abb85aec5f5687f1af373b6bb77995";
              var s = document.getElementsByTagName("script")[0]; 
              s.parentNode.insertBefore(hm, s);
            })();
            `,
          },
          {
            tagName: 'meta',
            attributes: {
              name: 'baidu-site-verification',
              content: 'codeva-ZZwz0GGvkQ',
            },
          },
        ],
      };
    },
  }
}
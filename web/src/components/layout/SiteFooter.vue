<template>
  <footer class="site-footer">
    <div class="container footer-inner">
      <div class="footer-top">
        <div class="footer-brand">
          <p class="footer-name">{{ site.settings.siteName || 'MyBlog' }}</p>
          <p class="footer-desc">{{ site.settings.siteDescription || '记录、思考与分享' }}</p>
        </div>
        <nav class="footer-col" aria-label="页脚导航">
          <p class="footer-head">导航</p>
          <RouterLink to="/">首页</RouterLink>
          <RouterLink to="/archives">归档</RouterLink>
          <RouterLink to="/categories">分类</RouterLink>
          <RouterLink to="/tags">标签</RouterLink>
          <RouterLink to="/series">专题</RouterLink>
          <RouterLink to="/links">友链</RouterLink>
        </nav>
        <div class="footer-col">
          <p class="footer-head">订阅</p>
          <a href="/rss" target="_blank" rel="noopener noreferrer">RSS 订阅</a>
          <a href="/sitemap.xml" target="_blank" rel="noopener noreferrer">网站地图</a>
        </div>
      </div>
      <div class="footer-meta">
        <span>{{ copyright }}</span>
        <a
          v-if="site.settings.icp"
          href="https://beian.miit.gov.cn/"
          target="_blank"
          rel="noopener noreferrer"
        >
          {{ site.settings.icp }}
        </a>
        <span>Powered by MyBlog</span>
      </div>
    </div>
  </footer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useSiteStore } from '@/stores/site'

const site = useSiteStore()

const copyright = computed(
  () =>
    site.settings.footerText ||
    `© ${new Date().getFullYear()} ${site.settings.siteName || 'MyBlog'}`
)
</script>

<style scoped>
/* 与页面同底色，仅一条 hairline 分隔——纸面延续；三区结构化（docs/05 §5.3） */
.site-footer {
  border-top: 1px solid var(--border);
  background: var(--bg);
}

.footer-inner {
  padding-top: 40px;
  padding-bottom: 28px;
}

.footer-top {
  display: grid;
  grid-template-columns: minmax(0, 2fr) 1fr 1fr;
  gap: 32px;
  padding-bottom: 32px;
  border-bottom: 1px solid var(--border);
}

.footer-name {
  font-family: var(--font-display);
  font-weight: var(--display-weight);
  font-size: 17px;
  line-height: 1.4;
}

.footer-desc {
  margin-top: 8px;
  max-width: 360px;
  font-size: 13.5px;
  color: var(--text-2);
  line-height: 1.8;
}

.footer-col {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 9px;
}

.footer-head {
  margin-bottom: 3px;
  font-family: var(--font-mono);
  font-size: 11px;
  font-weight: 600;
  letter-spacing: 0.14em;
  text-transform: uppercase;
  color: var(--text-3);
}

.footer-col a {
  font-size: 13.5px;
  color: var(--text-2);
  background-image: linear-gradient(var(--brand), var(--brand));
  background-size: 0% 1px;
  background-repeat: no-repeat;
  background-position: 0 100%;
  transition: color var(--transition), background-size 0.25s var(--ease-out-quart);
}

.footer-col a:hover {
  color: var(--brand);
  background-size: 100% 1px;
}

.footer-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 20px;
  padding-top: 20px;
  font-size: 12.5px;
  color: var(--text-3);
  font-variant-numeric: tabular-nums;
}

.footer-meta a {
  color: var(--text-3);
  transition: color var(--transition);
}

.footer-meta a:hover {
  color: var(--brand);
}

@media (max-width: 767px) {
  .footer-top {
    grid-template-columns: 1fr 1fr;
    gap: 28px;
  }

  .footer-brand {
    grid-column: 1 / -1;
  }

  .footer-meta {
    justify-content: center;
    text-align: center;
  }
}
</style>

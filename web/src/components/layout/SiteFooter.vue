<template>
  <footer class="site-footer">
    <div class="container footer-inner">
      <div class="footer-top">
        <div class="footer-brand">
          <p class="footer-name">
            <span class="footer-mark" aria-hidden="true">{{ brandInitial }}</span>
            <span>{{ site.settings.siteName || 'MyBlog' }}</span>
          </p>
          <p class="footer-desc">
            {{ site.settings.siteDescription || '记录 Go、Vue 与工程实践的长期笔记，写作即思考。' }}
          </p>
        </div>
        <nav class="footer-col" aria-label="页脚导航">
          <p class="footer-head">导航</p>
          <RouterLink to="/">首页</RouterLink>
          <RouterLink to="/archives">归档</RouterLink>
          <RouterLink to="/categories">分类</RouterLink>
          <RouterLink to="/tags">标签</RouterLink>
          <RouterLink to="/series">专题</RouterLink>
          <RouterLink to="/changelog">更新日志</RouterLink>
        </nav>
        <div class="footer-col">
          <p class="footer-head">订阅</p>
          <a href="/rss" target="_blank" rel="noopener noreferrer">RSS 订阅</a>
          <a href="/sitemap.xml" target="_blank" rel="noopener noreferrer">网站地图</a>
        </div>
        <div class="footer-col">
          <p class="footer-head">关于</p>
          <RouterLink to="/page/about">关于本站</RouterLink>
          <RouterLink to="/timeline">时间线</RouterLink>
          <RouterLink to="/links">友链</RouterLink>
        </div>
      </div>
      <div class="footer-meta">
        <span>{{ copyright }}</span>
        <a
          v-if="site.settings.icp"
          class="footer-icp"
          href="https://beian.miit.gov.cn/"
          target="_blank"
          rel="noopener noreferrer"
        >
          {{ site.settings.icp }}
        </a>
        <span class="footer-powered">Powered by MyBlog</span>
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
    `© ${new Date().getFullYear()} ${site.settings.siteName || 'MyBlog'} · 保留所有权利`
)

const brandInitial = computed(() =>
  (site.settings.siteName || 'MyBlog').trim().charAt(0).toUpperCase()
)
</script>

<style scoped>
/* 与页面同底色，仅一条 hairline 分隔；四栏结构化（p13 页脚） */
.site-footer {
  border-top: 1px solid var(--border);
  background: var(--bg);
}

.footer-inner {
  padding-top: 48px;
  padding-bottom: 28px;
}

.footer-top {
  display: grid;
  grid-template-columns: minmax(0, 1.8fr) 1fr 1fr 1fr;
  gap: 32px;
  padding-bottom: 36px;
}

.footer-name {
  display: flex;
  align-items: center;
  gap: 9px;
  font-family: var(--font-display);
  font-weight: var(--display-weight);
  font-size: var(--fs-lg);
  line-height: 1.4;
}

.footer-mark {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border-radius: var(--radius-sm);
  background: var(--brand);
  color: #fff;
  font-family: var(--font-sans);
  font-size: 14px;
  font-weight: 700;
  line-height: 1;
}

.footer-desc {
  margin-top: 10px;
  max-width: 320px;
  font-size: var(--fs-sm);
  color: var(--text-2);
  line-height: 1.85;
}

.footer-col {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 10px;
}

/* 栏头：sans 小字（p13 页脚为「导航 / 订阅 / 关于」普通小标题） */
.footer-head {
  margin-bottom: 2px;
  font-size: var(--fs-sm);
  font-weight: 600;
  color: var(--text-1);
}

.footer-col a {
  font-size: var(--fs-sm);
  color: var(--text-2);
  transition: color var(--transition);
}

.footer-col a:hover {
  color: var(--brand);
}

.footer-meta {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 20px;
  padding-top: 4px;
  font-size: var(--fs-xs);
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

.footer-icp {
  margin-left: auto;
}

@media (max-width: 767px) {
  .footer-top {
    grid-template-columns: 1fr 1fr;
    gap: 28px;
    padding-bottom: 28px;
  }

  .footer-brand {
    grid-column: 1 / -1;
  }

  .footer-icp {
    margin-left: 0;
  }
}
</style>

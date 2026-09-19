<template>
  <footer class="site-footer">
    <div class="container footer-inner">
      <p class="footer-line">{{ copyright }}</p>
      <p v-if="site.settings.icp" class="footer-line">
        <a href="https://beian.miit.gov.cn/" target="_blank" rel="noopener noreferrer">
          {{ site.settings.icp }}
        </a>
      </p>
      <p class="footer-line footer-meta">
        <a href="/rss" target="_blank" rel="noopener noreferrer">RSS 订阅</a>
        <span class="sep">·</span>
        <a href="/sitemap.xml" target="_blank" rel="noopener noreferrer">网站地图</a>
        <span class="sep">·</span>
        <span>Powered by MyBlog</span>
      </p>
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
.site-footer {
  border-top: 1px solid var(--border);
  background: var(--surface);
  margin-top: 24px;
}

.footer-inner {
  padding: 26px 20px;
  text-align: center;
}

.footer-line {
  font-size: 13px;
  color: var(--text-3);
  line-height: 2;
}

.footer-line a {
  color: var(--text-3);
  background-image: linear-gradient(var(--brand), var(--brand));
  background-size: 0% 1px;
  background-repeat: no-repeat;
  background-position: 0 100%;
  transition: color var(--transition), background-size 0.25s var(--ease-out-quart);
}

.footer-line a:hover {
  color: var(--brand);
  background-size: 100% 1px;
}

.footer-meta {
  margin-top: 2px;
}

.sep {
  margin: 0 6px;
}
</style>

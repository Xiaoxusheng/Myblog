<template>
  <div class="container not-found">
    <p v-reveal class="code">404</p>
    <p v-reveal="{ delay: 80 }" class="title">页面不存在或已被移动</p>
    <p v-reveal="{ delay: 140 }" class="desc">你访问的地址可能输错了，或者内容已经被删除。</p>
    <div v-reveal="{ delay: 200 }" class="actions">
      <RouterLink class="btn btn-primary" to="/">返回首页</RouterLink>
      <RouterLink class="btn btn-ghost" to="/archives">去归档看看</RouterLink>
    </div>
  </div>
</template>

<style scoped>
.not-found {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding-top: 96px;
  padding-bottom: 96px;
  text-align: center;
}

.code {
  font-family: var(--font-display);
  font-weight: var(--display-weight);
  font-size: clamp(72px, 12vw, 96px);
  line-height: 1;
  letter-spacing: -0.01em;
  color: var(--border-strong);
  font-variant-numeric: tabular-nums;
  user-select: none;
}

/* 与 Hero 同签名：尾部 accent 句点 */
.code::after {
  content: '.';
  color: var(--brand);
}

.title {
  margin-top: 20px;
  font-size: 19px;
  font-weight: 600;
  color: var(--text-1);
}

.desc {
  margin-top: 8px;
  font-size: 14px;
  color: var(--text-3);
}

.actions {
  display: flex;
  gap: 12px;
  margin-top: 26px;
}
</style>

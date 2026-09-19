<template>
  <div class="list-skeleton" aria-hidden="true">
    <div v-for="i in count" :key="i" class="card sk-card">
      <div class="sk-main">
        <span class="skeleton sk-item w30"></span>
        <span class="skeleton sk-item w80 tall"></span>
        <span class="skeleton sk-item w100"></span>
        <span class="skeleton sk-item w55"></span>
      </div>
      <span class="skeleton sk-thumb"></span>
    </div>
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ count?: number }>(), { count: 3 })
</script>

<style scoped>
.list-skeleton {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.sk-card {
  display: flex;
  gap: 20px;
  padding: 20px;
}

.sk-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.sk-item {
  display: block;
  height: 14px;
}

.sk-item.tall {
  height: 20px;
}

.w30 {
  width: 30%;
}

.w55 {
  width: 55%;
}

.w80 {
  width: 80%;
}

.w100 {
  width: 100%;
}

.sk-thumb {
  flex-shrink: 0;
  align-self: center;
  width: 168px;
  height: 112px;
  border-radius: var(--radius-sm);
}

@media (max-width: 640px) {
  .sk-card {
    flex-direction: column-reverse;
    gap: 12px;
    padding: 16px;
  }

  .sk-thumb {
    width: 100%;
    height: 140px;
  }
}
</style>

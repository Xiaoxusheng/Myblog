<template>
  <Teleport to="body">
    <Transition name="lightbox">
      <div v-if="src" class="lightbox-mask" role="dialog" aria-modal="true" aria-label="图片预览" @click="emit('close')">
        <img :src="src" :alt="alt" class="lightbox-img" @click.stop />
        <button class="lightbox-close" type="button" aria-label="关闭预览" @click.stop="emit('close')">
          <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" aria-hidden="true">
            <path d="M18 6L6 18M6 6l12 12" />
          </svg>
        </button>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { watch, onBeforeUnmount } from 'vue'

const props = defineProps<{ src: string; alt?: string }>()
const emit = defineEmits<{ close: [] }>()

function onKeydown(e: KeyboardEvent): void {
  if (e.key === 'Escape') emit('close')
}

// 打开时锁滚动 + 监听 Esc，关闭时还原
watch(
  () => props.src,
  (open) => {
    if (open) {
      document.body.style.overflow = 'hidden'
      window.addEventListener('keydown', onKeydown)
    } else {
      document.body.style.overflow = ''
      window.removeEventListener('keydown', onKeydown)
    }
  }
)

onBeforeUnmount(() => {
  document.body.style.overflow = ''
  window.removeEventListener('keydown', onKeydown)
})
</script>

<style scoped>
.lightbox-mask {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 32px;
  background: rgba(0, 0, 0, 0.72);
  cursor: zoom-out;
}

.lightbox-img {
  max-width: min(1100px, 94vw);
  max-height: 88vh;
  border-radius: 10px;
  box-shadow: 0 24px 64px rgba(0, 0, 0, 0.4);
  cursor: default;
}

.lightbox-close {
  position: absolute;
  top: 18px;
  right: 18px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border: none;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.14);
  color: #fff;
  cursor: pointer;
  transition: background 0.15s ease-out;
}

.lightbox-close:hover {
  background: rgba(255, 255, 255, 0.28);
}

.lightbox-enter-active,
.lightbox-leave-active {
  transition: opacity 0.18s ease-out;
}

.lightbox-enter-from,
.lightbox-leave-to {
  opacity: 0;
}
</style>

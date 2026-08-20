<template>
  <div
    class="flex items-center rounded-xl"
    :class="noHover ? '' : 'hover:border-accenthover:shadow-sm hover:bg-surface'"
  >
    <label
      class="flex cursor-pointer items-center gap-4 rounded-lg border border-border transition-all duration-200 ease-apple hover:border-accent"
      :class="{
        'border-none': noBorder,
        'p-4': !tighter,
      }"
    >
      <input v-model="modelValue" type="checkbox" class="peer hidden" @change.stop="emit('change', modelValue)" />
      <span
        class="relative shrink-0 rounded-full bg-gray-400/80 shadow-sm transition-all duration-medium ease-standard peer-checked:bg-accent peer-checked:shadow-[inset_0_1px_3px_rgba(0,0,0,0.1),0_2px_8px_color-mix(in_srgb,var(--color-accent),transparent_30%)] before:absolute before:rounded-full before:bg-white before:shadow-sm before:transition-all before:duration-200 before:ease-apple before:content-[''] peer-checked:before:translate-x-6"
        :class="
          small
            ? 'h-5.25 w-11 before:top-0.5 before:left-0.5 before:h-4.25 before:w-4.25'
            : 'h-7 w-13 before:top-0.75 before:left-0.75 before:h-5.5 before:w-5.5'
        "
      />
      <div class="flex flex-row items-center gap-1">
        <slot name="custom-title"></slot>
        <div v-if="!!title" class="flex flex-1 flex-col gap-1">
          <div>
            <span class="text-[0.925rem] leading-[1.4] font-semibold text-secondary">{{ title }}</span>
            <span v-if="required" class="ml-1 text-danger">*</span>
          </div>
          <span v-if="!!description" class="text-xs text-secondary/90">{{ description }}</span>
        </div>
        <slot name="switch-text"></slot>
      </div>
    </label>
    <slot name="title-extra"></slot>
    <div v-if="tips" class="group relative inline-block">
      <div
        class="flex h-5 w-5 cursor-pointer items-center justify-center rounded-full p-0.5 text-secondary hover:bg-bg-secondary hover:text-accent"
      >
        <Info :size="16" />
      </div>
      <div
        class="invisible absolute top-[125%] left-1/2 z-1000 w-max max-w-50 translate-x-[-50%] rounded-md border border-border bg-bg-tertiary p-2 text-center text-xs text-main opacity-0 shadow-md transition-opacity duration-300 group-hover:visible group-hover:opacity-100"
        v-html="transformMarkdownToHTML(tips)"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { Info } from 'lucide-vue-next'
import { marked } from 'marked'
import { onMounted } from 'vue'

const emit = defineEmits(['change'])

const modelValue = defineModel<boolean>()
const {
  title = '',
  description = '',
  noBorder = false,
  small = false,
  tips = '',
  required = false,
  noHover = false,
  tighter = false,
} = defineProps<{
  noBorder?: boolean
  title?: string
  description?: string
  small?: boolean
  tips?: string
  required?: boolean
  noHover?: boolean
  tighter?: boolean
}>()

function transformMarkdownToHTML(markdown: string) {
  try {
    return marked.parse(markdown)
  } catch (_e) {
    return markdown
  }
}

onMounted(() => {
  if (typeof modelValue.value === 'string') {
    modelValue.value = modelValue.value === 'true'
  }
})
</script>

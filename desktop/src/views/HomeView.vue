<template>
  <div class="relative flex h-full w-full items-center justify-center">
    <div class="relative z-1 flex h-full w-full items-center rounded-xl border-none p-4 overflow-hidden">
      <div class="w-full no-scrollbar h-full overflow-auto rounded-xl flex flex-col gap-8 items-center">
        <div class="flex w-full items-center justify-center border border-border rounded-xl gap-2 shadow-md">
          <QuickActionsCard />
        </div>
        <div class="flex max-md:flex-col w-full gap-3">
          <div class="flex w-full items-center justify-center border border-border rounded-xl gap-2 shadow-md">
            <CoreMonitorCard />
          </div>
          <div class="flex w-full items-center justify-center border border-border rounded-xl gap-2 shadow-md">
            <VersionManagerCard />
          </div>
        </div>
        <div
          class="flex no-scrollbar w-full items-center justify-center border border-border rounded-xl gap-2 shadow-md"
        >
          <DocumentationCard />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import CoreMonitorCard from '../components/dashboard/CoreMonitorCard.vue'
import DocumentationCard from '../components/dashboard/DocumentationCard.vue'
import QuickActionsCard from '../components/dashboard/QuickActionsCard.vue'
import VersionManagerCard from '../components/dashboard/VersionManagerCard.vue'
</script>

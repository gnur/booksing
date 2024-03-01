<script setup lang="ts">
const query = ref('')

const books = ref([])


watch(query, async () => {
  let q = query.value
  if (q.length < 2) return []

  const response = await fetch('/api/search?q=' + q)
  const data = await response.json()
  books.value = data.Items
})
</script>


<template>
  <UContainer>
    <div class="mt-10">
      <UInput placeholder="enter search terms..." size="xl" v-model="query" />
    </div>
    <div class="mt-10">
      <div class="grid grid-cols-3 gap-4">
        <div class="rounded border" v-for="book in books" :key="book.Hash">
          <div class="grid grid-cols-2">
            <div>
              <!-- TODO: when devproxy properly works
                 <img v-if="book.HasCover" src="/cover?hash={{book.Hash}}&file={{book.CoverPath}}" />
                 -->
            </div>
            <div>
              <div class="font-bold">{{ book.Title }}</div>
              <div>{{ book.Author }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </UContainer>
</template>

<script setup lang="ts">
const query = ref('')

const books = ref([])


watch(query, async () => {
  let q = query.value
  if (q.length < 2) {
    books.value = []
    return
  }

  const response = await fetch('/api/search?q=' + q)
  const data = await response.json()
  if (data.Items == undefined || data.Items.length == 0) {
    books.value = []
    return
  }
  books.value = data.Items
})

function coverPath(file: string) {
  return '/cover?file=' + file
}
</script>


<template>
  <UContainer>
    <div class="mt-10 sticky top-0 p-8">
      <UInput placeholder="enter search terms..." size="xl" v-model="query" />
    </div>



    <div class="mt-10">
      <div v-if="books.length == 0" class="text-center">
        <div v-if="query.length > 1" class="text-2xl">
          <h1 class="text-2xl">No books found</h1>
        </div>
        <div v-else class="text-2xl">
          <h1 class="text-2xl">Enter search terms to find books</h1>
        </div>
      </div>
      <div class="grid grid-cols-3 gap-4">
        <div class="rounded border" v-for="book in books" :key="book.Hash">
          <div class="grid grid-cols-2 h-full">
            <div class="p-3 flex items-center text-justify w-full">
              <img v-if="book.HasCover && book.CoverPath != ''" height="300px" :src="coverPath(book.CoverPath)" />
              <img v-else height="300px" src="/book.png" />
            </div>
            <div class="flex items-center justify-center">
              <span>
                <h1 class="font-bold">{{ book.Title }}</h1>
                <h2>{{ book.Author }}</h2>
                <a :href="'/download?hash=' + book.Hash" class="text-blue-500">DL</a>
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </UContainer>
</template>

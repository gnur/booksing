<script setup lang="ts">

const books = ref([])
const isOpen = ref(false)
const route = useRoute()
const router = useRouter()
const results = ref(0)

const selectedBook = ref({})

const total = await $fetch('/api/count', {
  method: 'GET',
  headers: {
    'Content-Type': 'application/json'
  }
})
console.log(total)

useHead({
  title: 'booksing'
})
const query = ref(route.query.search ? route.query.search : '')
const offset = ref(route.query.offset ? route.query.offset : 0)
let lastSearch = query.value.toString()

watch([query, offset], async () => {
  if (query.value != lastSearch || parseInt(offset.value.toString()) < 0) {
    offset.value = 0
  }
  lastSearch = query.value.toString()
  if (query.value != '') {
    router.push({ query: { search: query.value, offset: offset.value } })
  }
  let q = query.value
  if (q.length < 2) {
    router.push({})
    books.value = []
    results.value = 0
    return
  }

  const response = await fetch('/api/search?q=' + q + '&o=' + offset.value, {
    method: 'GET',
    headers: {
      'Content-Type': 'application/json'
    }
  })
  const data = await response.json()
  if (data.Items == undefined || data.Items.length == 0) {
    books.value = []
    return
  }
  books.value = data.Items
  results.value = data.Total
},
  { immediate: true })

function coverPath(file: string) {
  return '/api/cover?file=' + file
}

function downloadLink(file: string) {
  return '/api/download?hash=' + file
}

function showDetails(book: any) {
  isOpen.value = true
  selectedBook.value = book
}

function fileName(file: string) {
  return file.split('/').pop()
}

function nl2br(str: string) {
  return (str + '').replace(/([^>\r\n]?)(\r\n|\n\r|\r|\n)/g, '$1<br>$2');
}
</script>

<style lang="css">
.list-enter-active,
.list-leave-active {
  transition: all 0.5s ease;
}

.list-enter-from,
.list-leave-to {
  opacity: 0;
}
</style>

<template>
  <UContainer>
    <div class="mt-10 sticky top-0 p-8 backdrop-brightness-125 rounded-3xl">
      <UInput placeholder="enter search terms..." size="xl" v-model="query" />

      <div class="grid grid-cols-3 mt-5">
        <div>
          <UButton @click="offset = offset - 9" v-if="offset > 0">prev</UButton>
        </div>
        <div class="flex justify-center">
          <span v-if="results > 0">{{ results }} books found</span>
        </div>
        <div class="flex justify-end">
          <UButton @click="offset += 9" v-if="(offset + 9) < results">next</UButton>
        </div>
      </div>
    </div>



    <div class="mt-10">
      <div v-if="books.length == 0" class="text-center">
        <div v-if="query.length > 1" class="text-2xl">
          <h1 class="text-2xl">No books found</h1>
        </div>
        <div v-else class="text-2xl">
          <h1 class="text-4xl">Enter search terms to find books</h1>
          <h2 class="text-xl">Currently serving {{ total.total }} books</h2>
        </div>
      </div>
      <TransitionGroup name="list" tag="div" class="grid grid-cols-2 md:grid-cols-3 gap-4">
        <div class="rounded-lg border" v-for="book in books" :key="book.Hash">
          <div class="grid grid-cols-2 h-full" @click="showDetails(book)">
            <div class="p-3 flex items-center text-justify w-full">
              <img v-if="book.HasCover && book.CoverPath != ''" height="300px" :src="coverPath(book.CoverPath)" />
              <img v-else height="300px" src="/book.png" />
            </div>
            <div class="flex items-center justify-center">
              <span>
                <h1 class="font-bold">{{ book.Title }}</h1>
                <h2>{{ book.Author }}</h2>
              </span>
            </div>
          </div>
        </div>
      </TransitionGroup>
    </div>
  </UContainer>
  <USlideover v-model="isOpen">

    <UCard class="flex flex-col flex-1"
      :ui="{ body: { base: 'flex-1' }, ring: '', divide: 'divide-y divide-gray-100 dark:divide-gray-800' }">
      <template #header>
        <div class="flex items-center justify-between">
          <h3 class="text-base font-semibold leading-6 text-gray-900 dark:text-white">
            details
          </h3>
          <UButton color="gray" variant="ghost" icon="i-heroicons-x-mark-20-solid" class="-my-1"
            @click="isOpen = false" />
        </div>
      </template>


      <div class="flex-1">
        <h1 class="text-2xl">{{ selectedBook.Title }}</h1>
        <h2 class="text-xl">{{ selectedBook.Author }}</h2>

        <div class="p-5 mt-4 italic backdrop-brightness-125">
          <p v-html="nl2br(selectedBook.Description)"></p>
        </div>
        <div class="text-xs mt-7 ">
          {{ fileName(selectedBook.Path) }}
        </div>
      </div>


      <template #footer>
        <div class="flex justify-center">
          <a :href="downloadLink(selectedBook.Hash)" class="p-3 bg-blue-500 text-white">download</a>
        </div>
      </template>
    </UCard>

  </USlideover>
</template>

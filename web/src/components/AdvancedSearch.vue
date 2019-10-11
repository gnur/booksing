<template>
  <div id="app" class="container">
    <nav class="level">
      <b-field>
        <b-input
          placeholder="Search..."
          type="search"
          v-model="searchstring"
          id="search"
          size="is-medium"
          icon="magnify"
        ></b-input>
      </b-field>
      <button
        class="button field is-danger"
        @click="deleteSelectedBooks"
        v-if="isAdmin && checkedRows.length > 0"
      >
        <b-icon icon="delete"></b-icon>
        <span>Delete selected ({{ checkedRows.length }})</span>
      </button>
      <router-link v-if="isAdmin" :to="{ name: 'admin' }" class="button field is-info">admin</router-link>
    </nav>

    <div class="section">
      <b-table
        :data="books"
        paginated
        striped
        narrowed
        detailed
        :has-detailed-visible="showDetailed"
        :checked-rows.sync="checkedRows"
        :checkable="isAdmin"
        :loading="isLoading"
        per-page="50"
      >
        <template slot-scope="props">
          <b-table-column field="author" label="author">{{ props.row.author }}</b-table-column>
          <b-table-column field="title" label="title">
            <span v-if="longTitle(props.row.title)">
              <b-tooltip
                :label="props.row.title"
                type="is-light"
                :delay="500"
                dashed
              >{{ limitTitleLength(props.row.title) }}</b-tooltip>
            </span>
            <span v-else>{{ props.row.title }}</span>
          </b-table-column>
          <b-table-column field="language" label="language">{{ props.row.language }}</b-table-column>
          <b-table-column field="added" label="added">{{ formatDate(props.row.date_added) }}</b-table-column>
        </template>
        <template slot="detail" slot-scope="props">
          <article class="media">
            <figure class="media-left">
              <p class="image is-64x64"></p>
            </figure>
            <div class="media-content">
              <div class="content">
                <p>
                  <strong>{{ props.row.author }}</strong>
                  <small>&nbsp;{{ props.row.title }}</small>
                  <br />
                  <span v-html="formatFullMessage(props.row.description)" />
                </p>
              </div>
              <nav class="level is-mobile">
                <div class="level-left">
                  <template v-for="(v, index) in props.row.locations">
                    <a
                      :key="index"
                      class="level-item"
                      :href="'/auth/download?hash=' + props.row.hash + '&index=' + index"
                    >
                      <span>.{{ index}}</span>
                    </a>
                  </template>
                  <a
                    v-if="!hasMobi(props.row)"
                    class="level-item"
                    @click="convertBook(props.row.hash)"
                  >
                    <b-icon icon="refresh"></b-icon>
                    <span>create .mobi</span>
                  </a>
                </div>
              </nav>
            </div>
          </article>
          <br />
        </template>

        <template slot="empty">
          <section class="section">
            <div class="content has-text-grey has-text-centered">
              <p>
                <b-icon icon="emoticon-sad" size="is-large"></b-icon>
              </p>
              <p>Nothing here.</p>
            </div>
          </section>
        </template>
      </b-table>
    </div>
  </div>
</template>

<script>
import axios from "axios";
import lodash from "lodash";

export default {
  name: "home",
  data: function() {
    return {
      searchstring: "",
      books: [],
      total: 0,
      checkedRows: [],
      isLoading: true,
      isAdmin: false,
      refreshButtonText: "refresh"
    };
  },
  watch: {
    // whenever question changes, this function will run
    searchstring: function() {
      this.isLoading = true;
      this.getBooks();
    }
  },
  mounted: function() {
    this.getUser();
    this.getBooks();
  },

  methods: {
    formatFullMessage(description) {
      if (description == "") {
        return "No description.";
      }
      return (
        "<span>" +
        description.replace(/([^>\r\n]?)(\r\n|\n\r|\r|\n)/g, "$1<br>$2") +
        "</span>"
      );
    },
    hasMobi(book) {
      return "mobi" in book.locations;
    },
    longTitle(title) {
      return title.length > 53;
    },
    limitTitleLength(title) {
      if (title.length > 53) {
        return title.substring(0, 50) + "...";
      }
      return title;
    },
    formatDate(dateStr) {
      var d = new Date(dateStr);
      return d.toLocaleDateString("nl-NL", {
        year: "numeric",
        month: "long",
        day: "numeric"
      });
    },
    showDetailed(book) {
      return true;
    },
    convertBook: function(hash) {
      console.log(hash);
      var vm = this;
      vm.isLoading = true;
      const params = new URLSearchParams();
      params.append("hash", hash);
      axios
        .post("/auth/convert", params)
        .then(function(response) {
          vm.getBooks();
          console.log(response);
        })
        .catch(function(error) {
          console.log(error);
        });
    },
    getBooks: lodash.debounce(
      function() {
        var vm = this;
        vm.statusMessage = "getting results";
        var uri = "/auth/search";
        axios
          .get(uri, {
            params: {
              filter: this.searchstring,
              results: 500
            }
          })
          .then(function(response) {
            vm.books = response.data.books;
            if (vm.books === null) {
              vm.books = [];
            }
            vm.total = response.data.total;
            document.title = `booksing - ${vm.total} books available for searching`;
            vm.isLoading = false;
            vm.checkedRows = [];
          })
          .catch(function(error) {
            vm.statusMessage = "Something went wrong";
            console.log(error);
          });
      },
      // This is the number of milliseconds we wait for the
      // user to stop typing.
      500
    ),
    deleteSelectedBooks: function() {
      var vm = this;
      vm.isLoading = true;
      for (var book of vm.checkedRows) {
        const params = new URLSearchParams();
        params.append("hash", book.hash);
        axios
          .post("/admin/delete", params)
          .then(function(response) {
            vm.getBooks();
          })
          .catch(function(error) {
            console.log(error);
          });
      }
    },
    refreshBooklist: function() {
      var vm = this;
      vm.refreshButtonText = "Refreshing...";
      axios
        .get("/admin/refresh")
        .then(function(response) {
          vm.refreshButtonText = "refresh";
          vm.getBooks();
        })
        .catch(function(error) {
          vm.refreshButtonText = "refresh";
          console.log(error);
        });
    },
    getUser: function() {
      var vm = this;
      axios
        .get("/auth/user.json")
        .then(function(response) {
          vm.isAdmin = response.data.admin;
        })
        .catch(function(error) {
          console.log(error);
        });
    }
  }
};
</script>

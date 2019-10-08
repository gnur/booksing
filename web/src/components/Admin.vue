<template>
  <section>
    <button class="button field is-info" @click="refreshBooklist" v-if="isAdmin">
      <b-icon icon="refresh"></b-icon>
      <span>{{ refreshButtonText }}</span>
    </button>
    <router-link :to="{ name: 'new' }" class="button field is-info">search</router-link>
    <b-tabs position="is-centered" class="block">
      <b-tab-item label="downloads">
        <b-table :data="downloads" paginated striped narrowed per-page="50">
          <template slot-scope="props">
            <b-table-column
              field="timestamp"
              label="timestamp"
            >{{ formatDate(props.row.timestamp) }}</b-table-column>
            <b-table-column field="user" label="user">{{ props.row.user }}</b-table-column>
            <b-table-column field="book" label="book">{{ props.row.hash }}</b-table-column>
          </template>
        </b-table>
      </b-tab-item>
      <b-tab-item label="refreshes">
        <b-table :data="refreshes" paginated striped narrowed per-page="50">
          <template slot-scope="props">
            <b-table-column field="starttime" label="start">{{ formatDate(props.row.StartTime) }}</b-table-column>
            <b-table-column
              field="runtime"
              label="runtime"
            >{{ getDuration(props.row.StartTime, props.row.StopTime) }}</b-table-column>
            <b-table-column field="old" label="old">{{ props.row.Old }}</b-table-column>
            <b-table-column field="added" label="added">{{ props.row.Added }}</b-table-column>
            <b-table-column field="duplicate" label="duplicate">{{ props.row.Duplicate }}</b-table-column>
            <b-table-column field="invalid" label="invalid">{{ props.row.Invalid }}</b-table-column>
          </template>
        </b-table>
      </b-tab-item>
    </b-tabs>
  </section>
</template>

<script>
import axios from "axios";
import lodash from "lodash";

export default {
  name: "home",
  data: function() {
    return {
      downloads: [],
      refreshes: [],
      isAdmin: false,
      refreshButtonText: "refresh"
    };
  },
  mounted: function() {
    this.getDownloads();
    this.getRefreshes();
    this.getUser();
  },

  methods: {
    formatDate(dateStr) {
      var d = new Date(dateStr);
      var input = new Date(dateStr);
      var today = new Date();
      if (d.setHours(0, 0, 0, 0) == today.setHours(0, 0, 0, 0)) {
        return input.toLocaleTimeString("nl-NL", {});
      } else {
        return input.toLocaleString("nl-NL", {});
      }
    },

    refreshBooklist: function() {
      var vm = this;
      vm.refreshButtonText = "Refreshing...";
      axios
        .get("/auth/refresh")
        .then(function(response) {
          vm.refreshButtonText = "refresh";
          vm.getRefreshes();
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
    },
    formatDateTime(dateStr) {
      return dateStr.replace("T", " ").substr(0, 16);
    },
    getDuration(startDateStr, endDateStr) {
      var date1 = new Date(startDateStr);
      var date2 = new Date(endDateStr);

      var msec = date2.getTime() - date1.getTime();

      var returnstring = "";
      var hh = Math.floor(msec / 1000 / 60 / 60);
      if (hh > 0) {
        returnstring += h + "h";
      }
      msec -= hh * 1000 * 60 * 60;
      var mm = Math.floor(msec / 1000 / 60);
      if (mm > 0) {
        returnstring += mm + "m";
      }
      msec -= mm * 1000 * 60;
      var ss = Math.floor(msec / 1000);
      if (ss > 0) {
        returnstring += ss + "s";
      }
      msec -= ss * 1000;
      if (ss == 0 && msec > 0) {
        returnstring += msec + "ms";
      }

      return returnstring;
    },
    getDownloads: function() {
      var vm = this;
      axios
        .get("/admin/downloads.json")
        .then(function(response) {
          if (response.data !== null) {
            vm.downloads = response.data;
          } else {
            vm.downloads = [];
          }
        })
        .catch(function(error) {
          console.log(error);
        });
    },
    getRefreshes: function() {
      var vm = this;
      axios
        .get("/admin/refreshes.json")
        .then(function(response) {
          vm.refreshes = response.data;
        })
        .catch(function(error) {
          console.log(error);
        });
    }
  }
};
</script>

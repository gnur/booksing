<template>
  <div class="container">
    <article v-if="showWarning" class="message is-warning">
      <div class="message-header">
        <p>{{ warningMessage }}</p>
      </div>
    </article>
    <article v-if="showSuccess" class="message is-success">
      <div class="message-header">
        <p>{{ successMessage }}</p>
      </div>
    </article>
    <b-table :data="apikeys">
      <template slot-scope="props">
        <b-table-column field="id" label="id">
          {{
          props.row.id
          }}
        </b-table-column>
        <b-table-column field="created" label="created">
          {{
          formatDate(props.row.Created)
          }}
        </b-table-column>
        <b-table-column field="last use" label="last use">
          {{
          formatDate(props.row.LastUsed)
          }}
        </b-table-column>
        <b-table-column field="id" label="delete">
          <a @click="confirmDelete(props.row.Key)">
            <span class="icon">
              <i class="mdi mdi-delete"></i>
            </span>
          </a>
        </b-table-column>
      </template>
    </b-table>
    <!-- add api key code -->
    <div class="field has-addons">
      <div class="control">
        <input v-model="apikeyid" class="input" type="text" placeholder="API user" />
      </div>
      <div class="control">
        <a class="button is-info" @click="addAPIKey">save</a>
      </div>
    </div>
    <!-- end api key code -->
  </div>
</template>

<script>
import axios from "axios";
import router from "../router";
// Firebase App (the core Firebase SDK) is always required and must be listed first
import * as firebase from "firebase/app";

import "firebase/auth";

// platform
// direct
export default {
  name: "Dashboard",
  components: {},
  props: {},
  data() {
    return {
      apikeys: [],
      apikeyid: "",
      showWarning: false,
      showSuccess: false,
      warningMessage: "",
      successMessage: "",
      username: this.$store.getters.username
    };
  },

  mounted: function() {
    axios.defaults.headers.common["Authorization"] =
      "Bearer " + this.$store.getters.token;
    this.refresh();
  },
  methods: {
    refresh() {
      axios
        .get("/auth/apikey")
        .then(resp => {
          if (resp.data.user.APIKeys != null) {
            this.apikeys = resp.data.user.APIKeys;
          } else {
            this.apikeys = [];
          }
        })
        .catch(function(error) {
          console.log(error);
          if (error.response && error.response.status == 403) {
            router.push({ name: "login" });
          }
        });
    },
    confirmDelete: function(uuid) {
      this.$dialog.confirm({
        message: "Delete this api key",
        type: "is-danger",
        hasIcon: true,
        onConfirm: () => this.deleteAPIKey(uuid)
      });
    },
    deleteAPIKey: function(uuid) {
      axios.delete("/auth/apikey/" + encodeURIComponent(uuid)).then(
        () => {
          this.refresh();
          this.$toast.open({
            duration: 2000,
            type: "is-success",
            message: "key deleted",
            position: "is-bottom"
          });
        },
        err => {
          this.$toast.open({
            duration: 3000,
            type: "is-danger",
            message: "note failed to delete: " + err,
            position: "is-bottom"
          });
          console.log(err);
        }
      );
    },
    addAPIKey() {
      var vm = this;
      axios
        .post("/auth/apikey", {
          id: this.apikeyid
        })
        .then(resp => {
          this.apikeyid = "";
          this.showSuccessMsg("api key added: " + resp.data.key.Key);
          this.refresh();
        })
        .catch(err => {
          this.showErrorAlert("failed adding api key: " + err);
        });
    },
    formatDate(dateStr) {
      var d = new Date(dateStr);
      return d.toLocaleDateString("nl-NL", {
        year: "numeric",
        month: "long",
        day: "numeric"
      });
    },
    showSuccessMsg: function(msg) {
      this.successMessage = msg;
      this.showSuccess = true;
    },

    showErrorAlert: function(msg) {
      this.warningMessage = msg;
      this.showWarning = true;
    },

    hideErrorAlert: function() {
      this.showWarning = false;
    }
  }
};
</script>

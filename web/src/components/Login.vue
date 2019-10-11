<template>
  <div class="hero is-fullheight is-dark is-bold">
    <div class="hero-body">
      <div class="container">
        <h1 class="title has-text-centered">login</h1>
        <div class="box">
          <a v-if="!loggedin" @click="googleLogin">login with google</a>
          <a v-if="loggedin">continue</a>
          <b-loading :is-full-page="isFullPage" :active.sync="loading"></b-loading>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import axios from "axios";
import router from "../router";
import store from "../store";
// Firebase App (the core Firebase SDK) is always required and must be listed first
import * as firebase from "firebase/app";

import "firebase/auth";

export default {
  name: "Login",
  components: {},

  props: {
    msg: {
      type: String,
      default: ""
    }
  },
  data() {
    return {
      input: {
        username: "",
        password: ""
      },
      loggedin: false,
      isFullPage: false,
      loading: true,
      showWarning: false,
      warningMessage: ""
    };
  },
  mounted: function() {
    console.log("starting authchangestatefunction");
    firebase.auth().onAuthStateChanged(user => {
      console.log("callback authchangestatefunction");
      if (user) {
        // User is signed in.
        user.getIdToken(true).then(idToken => {
          store.dispatch("login", {
            username: user.email,
            token: idToken
          });
          router.push({ name: "home" });
        });
      } else {
        this.loading = false;
        this.loggedin = false;
        // No user is signed in.
      }
    });
  },
  methods: {
    googleLogin: function() {
      var provider = new firebase.auth.GoogleAuthProvider();

      firebase
        .auth()
        .signInWithPopup(provider)
        .then(result => {
          // This gives you a Google Access Token. You can use it to access the Google API.
          var token = result.credential.accessToken;
          var idToken = result.credential.idToken;
          // The signed-in user info.
          console.dir(result);
          var user = result.user;
          firebase
            .auth()
            .currentUser.getIdToken(/* forceRefresh */ true)
            .then(function(idToken) {
              axios
                .post("/checkToken", { idToken: idToken })
                .then(resp => {
                  store.dispatch("login", user.email);
                  router.push({ name: "home" });
                })
                .catch(err => {
                  console.log(err);
                  this.showErrorAlert(err);
                });
            });
        })
        .catch(function(error) {
          console.log("ERROR");
          // Handle Errors here.
          var errorCode = error.code;
          var errorMessage = error.message;
          // The email of the user's account used.
          var email = error.email;
          // The firebase.auth.AuthCredential type that was used.
          var credential = error.credential;
          // ...
          console.log(errorCode);
          console.log(errorMessage);
          console.log(email);
          console.log(credential);
          this.showErrorAlert(errorMessage);
          // ...
        });
    },

    showErrorAlert: function(msg) {
      this.$toast.open({
        duration: 5000,
        message: msg,
        type: "is-danger"
      });
    }
  }
};
</script>

<style>
.loginform {
  width: 640px;
}
</style>

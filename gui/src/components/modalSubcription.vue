<template>
  <div class="modal-card" style="max-width: 400px; margin: auto">
    <header class="modal-card-head">
      <p class="modal-card-title">{{ $t("configureSubscription.title") }}</p>
    </header>
    <section class="modal-card-body">
      <b-field label="SUBSCRIPTION">
        <b-input
          v-model="which.address"
          type="textarea"
          :placeholder="$t('subscription.subscription')"
        />
      </b-field>
      <b-field label="REMARKS">
        <b-input
          v-model="which.remarks"
          :placeholder="$t('subscription.remarks')"
        />
      </b-field>
      <b-field label="AUTO-SELECT">
        <b-checkbox
          v-model="which.autoSelect"
        >{{ $t("subscription.autoSelect") }}
        </b-checkbox>
      </b-field>
      <b-field label="OUTBOUND GROUPS">
        <b-select v-model="selectedOutbounds" multiple expanded native-size="4">
          <option v-for="outbound in normalizedOutbounds" :key="outbound" :value="outbound">
            {{ outbound }}
          </option>
        </b-select>
      </b-field>
    </section>
    <footer class="modal-card-foot flex-end">
      <button class="button" type="button" @click="$parent.close()">
        {{ $t("operations.cancel") }}
      </button>
      <button class="button is-primary" @click="handleClickSubmit">
        {{ $t("operations.saveApply") }}
      </button>
    </footer>
  </div>
</template>

<script>
export default {
  name: "ModalSubscription",
  props: {
    which: {
      type: Object,
      default() {
        return null;
      },
    },
    outbounds: {
      type: Array,
      default() {
        return ["proxy"];
      },
    },
  },
  data() {
    return {
      selectedOutbounds: [],
    };
  },
  computed: {
    normalizedOutbounds() {
      const seen = new Set();
      const result = [];
      for (const outbound of this.outbounds || []) {
        if (typeof outbound !== "string") {
          continue;
        }
        const name = outbound.trim();
        if (!name || seen.has(name)) {
          continue;
        }
        seen.add(name);
        result.push(name);
      }
      if (!seen.has("proxy")) {
        result.unshift("proxy");
      }
      return result;
    },
  },
  watch: {
    which: {
      immediate: true,
      handler(which) {
        const outbounds = which && Array.isArray(which.outbounds) && which.outbounds.length
          ? which.outbounds
          : ["proxy"];
        this.selectedOutbounds = outbounds.filter((x) => this.normalizedOutbounds.includes(x));
        if (this.selectedOutbounds.length === 0) {
          this.selectedOutbounds = ["proxy"];
        }
      },
    },
  },
  methods: {
    handleClickSubmit() {
      this.$emit("submit", {
        ...this.which,
        outbounds: this.selectedOutbounds.length ? this.selectedOutbounds : ["proxy"],
      });
    },
  },
};
</script>

<style lang="scss">
.is-twitter .is-active a {
  color: #4099ff !important;
}
.readonly {
  pointer-events: none;
}
.same-width-5 li {
  width: 5em;
}
</style>
